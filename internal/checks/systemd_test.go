package checks

import (
	"errors"
	"strings"
	"testing"

	"faultradar/internal/config"
	"faultradar/internal/model"
	"faultradar/internal/system"
)

func TestCheckSystemd(t *testing.T) {
	cfg := config.DefaultConfig()

	// 1. no failed units -> ok
	t.Run("no failed units -> ok", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\n\n0 loaded units listed.\n"), nil
			},
		}
		findings := CheckSystemd(runner, cfg)
		if len(findings) != 2 {
			t.Fatalf("expected 2 findings, got %d", len(findings))
		}
		if findings[0].Severity != model.SeverityOK {
			t.Errorf("expected OK for failed units, got %v", findings[0].Severity)
		}
		if findings[1].Severity != model.SeverityOK {
			t.Errorf("expected OK for snap mounts, got %v", findings[1].Severity)
		}
	})

	// 2. only snap mount failures -> warning on snap mounts, ok on units
	t.Run("only snap mount failures -> warning on snap mounts, ok on units", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\nsnap-chromium-3235.mount loaded failed failed Mount chromium snap\n"), nil
			},
		}
		findings := CheckSystemd(runner, cfg)
		if len(findings) != 2 {
			t.Fatalf("expected 2 findings, got %d", len(findings))
		}
		if findings[0].Severity != model.SeverityOK {
			t.Errorf("expected OK for standard failed units, got %v", findings[0].Severity)
		}
		if findings[1].Severity != model.SeverityWarning {
			t.Errorf("expected Warning for failed snap mounts, got %v", findings[1].Severity)
		}
		// check details contains snap mount units
		var hasSnapHeader bool
		for _, detail := range findings[1].Details {
			if strings.Contains(detail, "Failed snap mount units:") {
				hasSnapHeader = true
			}
		}
		if !hasSnapHeader {
			t.Errorf("expected snap finding details to contain Failed snap mount units group")
		}
	})

	// 3. real failed service -> warning on units, ok on snap mounts
	t.Run("real failed service -> warning on units, ok on snap mounts", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\ncups.service loaded failed failed CUPS Scheduler\n"), nil
			},
		}
		findings := CheckSystemd(runner, cfg)
		if len(findings) != 2 {
			t.Fatalf("expected 2 findings, got %d", len(findings))
		}
		if findings[0].Severity != model.SeverityWarning {
			t.Errorf("expected Warning, got %v", findings[0].Severity)
		}
		if findings[1].Severity != model.SeverityOK {
			t.Errorf("expected OK for snap mounts, got %v", findings[1].Severity)
		}
	})

	// 4. important failed service -> critical on units, ok on snap mounts
	t.Run("important failed service -> critical on units, ok on snap mounts", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\nmysql.service loaded failed failed MySQL Community Server\n"), nil
			},
		}
		findings := CheckSystemd(runner, cfg)
		if len(findings) != 2 {
			t.Fatalf("expected 2 findings, got %d", len(findings))
		}
		if findings[0].Severity != model.SeverityCritical {
			t.Errorf("expected Critical for important service failure, got %v", findings[0].Severity)
		}
	})

	// 5. exact ignored unit -> ignored
	t.Run("exact ignored unit -> ignored", func(t *testing.T) {
		customConfig := config.DefaultConfig()
		customConfig.Systemd.IgnoreUnits = []string{"cups.service"}

		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\ncups.service loaded failed failed CUPS Scheduler\n"), nil
			},
		}
		findings := CheckSystemd(runner, customConfig)
		if findings[0].Severity != model.SeverityOK {
			t.Errorf("expected OK when failed unit is exact ignored, got %v", findings[0].Severity)
		}
	})

	// 6. glob ignored unit pattern snap-*.mount -> ignored
	t.Run("glob ignored unit pattern -> ignored", func(t *testing.T) {
		customConfig := config.DefaultConfig()
		customConfig.Systemd.IgnoreUnitPatterns = []string{"snap-*.mount"}

		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\nsnap-chromium-3235.mount loaded failed failed Mount chromium snap\n"), nil
			},
		}
		findings := CheckSystemd(runner, customConfig)
		if findings[1].Severity != model.SeverityOK {
			t.Errorf("expected OK when failed snap mount matches glob ignore, got %v", findings[1].Severity)
		}
	})

	// 7. command missing -> skipped
	t.Run("command missing -> skipped", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return nil, errors.New("exec: \"systemctl\": executable file not found in $PATH")
			},
		}
		findings := CheckSystemd(runner, cfg)
		if len(findings) != 2 {
			t.Fatalf("expected 2 findings, got %d", len(findings))
		}
		if findings[0].Severity != model.SeveritySkipped {
			t.Errorf("expected Skipped, got %v", findings[0].Severity)
		}
		if findings[1].Severity != model.SeveritySkipped {
			t.Errorf("expected Skipped, got %v", findings[1].Severity)
		}
	})

	// 8. long snap list is summarized to 3 examples
	t.Run("long snap list is summarized to 3 examples", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				output := "UNIT LOAD ACTIVE SUB DESCRIPTION\n"
				for i := 1; i <= 15; i++ {
					output += "snap-chromium-32" + string(rune('0'+i)) + ".mount loaded failed failed Mount chromium snap\n"
				}
				return []byte(output), nil
			},
		}
		findings := CheckSystemd(runner, cfg)
		// Check that the details list has truncated snap mounts
		var count int
		var foundSummaryLine bool
		for _, detail := range findings[1].Details {
			if strings.Contains(detail, "- snap-chromium-") {
				count++
			}
			if strings.Contains(detail, "... and 12 more") {
				foundSummaryLine = true
			}
		}
		if count != 3 {
			t.Errorf("expected 3 listed snap items, got %d", count)
		}
		if !foundSummaryLine {
			t.Errorf("expected summary line detailing remaining snap counts")
		}
	})
}
