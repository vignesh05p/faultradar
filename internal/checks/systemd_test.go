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

	t.Run("no failed units", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\n\n0 loaded units listed.\n"), nil
			},
		}
		findings := CheckSystemd(runner, cfg)
		if len(findings) != 1 {
			t.Fatalf("expected 1 finding, got %d", len(findings))
		}
		if findings[0].ID != "systemd.failed.none" {
			t.Errorf("expected systemd.failed.none, got %s", findings[0].ID)
		}
		if findings[0].Severity != model.SeverityOK {
			t.Errorf("expected OK, got %v", findings[0].Severity)
		}
	})

	t.Run("only snap mount failures", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\nsnap-chromium-3235.mount loaded failed failed Mount chromium snap\n"), nil
			},
		}
		findings := CheckSystemd(runner, cfg)
		if len(findings) != 1 {
			t.Fatalf("expected 1 finding, got %d", len(findings))
		}
		if findings[0].ID != "systemd.failed.snap_mounts" {
			t.Errorf("expected systemd.failed.snap_mounts, got %s", findings[0].ID)
		}
		if findings[0].Severity != model.SeverityWarning {
			t.Errorf("expected Warning, got %v", findings[0].Severity)
		}
	})

	t.Run("normal service failure becomes warning", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\ncups.service loaded failed failed CUPS Scheduler\n"), nil
			},
		}
		findings := CheckSystemd(runner, cfg)
		if len(findings) != 1 {
			t.Fatalf("expected 1 finding, got %d", len(findings))
		}
		if findings[0].ID != "systemd.failed.services" {
			t.Errorf("expected systemd.failed.services, got %s", findings[0].ID)
		}
		if findings[0].Severity != model.SeverityWarning {
			t.Errorf("expected Warning, got %v", findings[0].Severity)
		}
	})

	t.Run("important service failure becomes critical", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\nmysql.service loaded failed failed MySQL Community Server\n"), nil
			},
		}
		findings := CheckSystemd(runner, cfg)
		if len(findings) != 1 {
			t.Fatalf("expected 1 finding, got %d", len(findings))
		}
		if findings[0].ID != "systemd.failed.important" {
			t.Errorf("expected systemd.failed.important, got %s", findings[0].ID)
		}
		if findings[0].Severity != model.SeverityCritical {
			t.Errorf("expected Critical, got %v", findings[0].Severity)
		}
	})

	t.Run("snap mount failures separated from important services", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\nmysql.service loaded failed failed MySQL\nsnap-chromium-3235.mount loaded failed failed Mount chromium snap\n"), nil
			},
		}
		findings := CheckSystemd(runner, cfg)
		if len(findings) != 2 {
			t.Fatalf("expected 2 findings, got %d", len(findings))
		}
		if findings[0].ID != "systemd.failed.important" {
			t.Errorf("expected important finding first, got %s", findings[0].ID)
		}
		if findings[1].ID != "systemd.failed.snap_mounts" {
			t.Errorf("expected snap mount finding, got %s", findings[1].ID)
		}
	})

	t.Run("exact ignored unit", func(t *testing.T) {
		customConfig := config.DefaultConfig()
		customConfig.Systemd.IgnoreUnits = []string{"cups.service"}

		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\ncups.service loaded failed failed CUPS Scheduler\n"), nil
			},
		}
		findings := CheckSystemd(runner, customConfig)
		if len(findings) != 1 || findings[0].ID != "systemd.failed.none" {
			t.Errorf("expected ignored unit to produce no failures, got %+v", findings)
		}
	})

	t.Run("glob ignored unit pattern", func(t *testing.T) {
		customConfig := config.DefaultConfig()
		customConfig.Systemd.IgnoreUnitPatterns = []string{"snap-*.mount"}

		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\nsnap-chromium-3235.mount loaded failed failed Mount chromium snap\n"), nil
			},
		}
		findings := CheckSystemd(runner, customConfig)
		if len(findings) != 1 || findings[0].ID != "systemd.failed.none" {
			t.Errorf("expected ignored snap mount to produce no failures, got %+v", findings)
		}
	})

	t.Run("systemctl unavailable becomes skipped", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return nil, errors.New("exec: \"systemctl\": executable file not found in $PATH")
			},
		}
		findings := CheckSystemd(runner, cfg)
		if len(findings) != 1 {
			t.Fatalf("expected 1 finding, got %d", len(findings))
		}
		if findings[0].ID != "systemd.unavailable" {
			t.Errorf("expected systemd.unavailable, got %s", findings[0].ID)
		}
		if findings[0].Severity != model.SeveritySkipped {
			t.Errorf("expected Skipped, got %v", findings[0].Severity)
		}
	})

	t.Run("long list truncation works", func(t *testing.T) {
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
		var count int
		var foundSummaryLine bool
		for _, detail := range findings[0].Details {
			if strings.Contains(detail, "- snap-chromium-") {
				count++
			}
			if strings.Contains(detail, "... and 13 more") {
				foundSummaryLine = true
			}
		}
		if count != 2 {
			t.Errorf("expected 2 listed snap items, got %d", count)
		}
		if !foundSummaryLine {
			t.Errorf("expected summary line detailing remaining snap counts")
		}
	})
}
