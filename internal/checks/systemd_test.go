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

	t.Run("no failed units", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\n\n0 loaded units listed.\n"), nil
			},
		}

		finding := CheckSystemd(runner, config)
		if finding.Severity != model.SeverityOK {
			t.Errorf("expected OK severity, got %v", finding.Severity)
		}
	})

	t.Run("failed units", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\ncups.service loaded failed failed CUPS Scheduler\n\n1 loaded units listed.\n"), nil
			},
		}

		finding := CheckSystemd(runner, config)
		if finding.Severity != model.SeverityWarning {
			t.Errorf("expected Warning severity, got %v", finding.Severity)
		}
		foundCups := false
		for _, detail := range finding.Details {
			if strings.Contains(detail, "cups.service") {
				foundCups = true
			}
		}
		if !foundCups {
			t.Errorf("expected details to contain cups.service")
		}
	})

	t.Run("ignored failed unit", func(t *testing.T) {
		customConfig := model.DefaultConfig()
		customConfig.Systemd.IgnoreUnits = []string{"cups.service"}

		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("UNIT LOAD ACTIVE SUB DESCRIPTION\ncups.service loaded failed failed CUPS Scheduler\n\n1 loaded units listed.\n"), nil
			},
		}

		finding := CheckSystemd(runner, customConfig)
		if finding.Severity != model.SeverityOK {
			t.Errorf("expected OK severity because the only failed unit was ignored, got %v", finding.Severity)
		}
	})

	t.Run("command missing", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return nil, errors.New("exec: \"systemctl\": executable file not found in $PATH")
			},
		}

		finding := CheckSystemd(runner, config)
		if finding.Severity != model.SeveritySkipped {
			t.Errorf("expected Skipped severity on missing command, got %v", finding.Severity)
		}
	})
}
