package checks

import (
	"errors"
	"strings"
	"testing"

	"faultradar/internal/model"
	"faultradar/internal/system"
)

func TestCheckSystemd(t *testing.T) {
	config := model.DefaultConfig()

	// 1. no failed units -> ok
	t.Run("no failed units -> ok", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\n\n0 loaded units listed.\n"), nil
			},
		}
		finding := CheckSystemd(runner, config)
		if finding.Severity != model.SeverityOK {
			t.Errorf("expected OK, got %v", finding.Severity)
		}
	})

	// 2. only snap mount failures -> warning
	t.Run("only snap mount failures -> warning", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\nsnap-chromium-3235.mount loaded failed failed Mount chromium snap\n"), nil
			},
		}
		finding := CheckSystemd(runner, config)
		if finding.Severity != model.SeverityWarning {
			t.Errorf("expected Warning, got %v", finding.Severity)
		}
		// check details contains snap mount units
		var hasSnapHeader bool
		for _, detail := range finding.Details {
			if strings.Contains(detail, "Failed snap mount units:") {
				hasSnapHeader = true
			}
		}
		if !hasSnapHeader {
			t.Errorf("expected details to contain Failed snap mount units group")
		}
	})

	// 3. real failed service -> warning
	t.Run("real failed service -> warning", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\ncups.service loaded failed failed CUPS Scheduler\n"), nil
			},
		}
		finding := CheckSystemd(runner, config)
		if finding.Severity != model.SeverityWarning {
			t.Errorf("expected Warning, got %v", finding.Severity)
		}
	})

	// 4. important failed service -> critical
	t.Run("important failed service -> critical", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\nmysql.service loaded failed failed MySQL Community Server\n"), nil
			},
		}
		finding := CheckSystemd(runner, config)
		if finding.Severity != model.SeverityCritical {
			t.Errorf("expected Critical for important service failure, got %v", finding.Severity)
		}
	})

	// 5. exact ignored unit -> ignored
	t.Run("exact ignored unit -> ignored", func(t *testing.T) {
		customConfig := model.DefaultConfig()
		customConfig.Systemd.IgnoreUnits = []string{"cups.service"}

		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\ncups.service loaded failed failed CUPS Scheduler\n"), nil
			},
		}
		finding := CheckSystemd(runner, customConfig)
		if finding.Severity != model.SeverityOK {
			t.Errorf("expected OK when failed unit is exact ignored, got %v", finding.Severity)
		}
	})

	// 6. glob ignored unit pattern snap-*.mount -> ignored
	t.Run("glob ignored unit pattern -> ignored", func(t *testing.T) {
		customConfig := model.DefaultConfig()
		customConfig.Systemd.IgnoreUnitPatterns = []string{"snap-*.mount"}

		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\nsnap-chromium-3235.mount loaded failed failed Mount chromium snap\n"), nil
			},
		}
		finding := CheckSystemd(runner, customConfig)
		if finding.Severity != model.SeverityOK {
			t.Errorf("expected OK when failed unit matches glob ignore, got %v", finding.Severity)
		}
	})

	// 7. command missing -> skipped
	t.Run("command missing -> skipped", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return nil, errors.New("exec: \"systemctl\": executable file not found in $PATH")
			},
		}
		finding := CheckSystemd(runner, config)
		if finding.Severity != model.SeveritySkipped {
			t.Errorf("expected Skipped, got %v", finding.Severity)
		}
	})

	// 8. long list is summarized
	t.Run("long list is summarized", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				output := "UNIT LOAD ACTIVE SUB DESCRIPTION\n"
				for i := 1; i <= 15; i++ {
					output += "snap-chromium-" + string(rune('0'+i)) + ".mount loaded failed failed Mount chromium snap\n"
				}
				return []byte(output), nil
			},
		}
		finding := CheckSystemd(runner, config)
		// Check that the details list has truncated snap mounts
		var count int
		var foundSummaryLine bool
		for _, detail := range finding.Details {
			if strings.Contains(detail, "- snap-chromium-") {
				count++
			}
			if strings.Contains(detail, "... and 5 more") {
				foundSummaryLine = true
			}
		}
		if count != 10 {
			t.Errorf("expected 10 listed items, got %d", count)
		}
		if !foundSummaryLine {
			t.Errorf("expected summary line detailing remaining counts")
		}
	})
}
