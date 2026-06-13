package checks

import (
	"errors"
	"strings"
	"testing"

	"faultradar/internal/model"
	"faultradar/internal/system"
)

func TestCheckKernel(t *testing.T) {
	config := model.DefaultConfig()

	t.Run("no errors", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte(""), nil
			},
		}

		finding := CheckKernel(runner, config)
		if finding.Severity != model.SeverityOK {
			t.Errorf("expected OK severity, got %v", finding.Severity)
		}
	})

	t.Run("warning count", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				lines := make([]string, 10)
				for i := range lines {
					lines[i] = "kernel error line"
				}
				return []byte(strings.Join(lines, "\n")), nil
			},
		}

		finding := CheckKernel(runner, config)
		if finding.Severity != model.SeverityWarning {
			t.Errorf("expected Warning severity, got %v", finding.Severity)
		}
	})

	t.Run("critical count", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				lines := make([]string, 30)
				for i := range lines {
					lines[i] = "kernel error line"
				}
				return []byte(strings.Join(lines, "\n")), nil
			},
		}

		finding := CheckKernel(runner, config)
		if finding.Severity != model.SeverityCritical {
			t.Errorf("expected Critical severity, got %v", finding.Severity)
		}
	})

	t.Run("command missing", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return nil, errors.New("exec: \"journalctl\": executable file not found in $PATH")
			},
		}

		finding := CheckKernel(runner, config)
		if finding.Severity != model.SeveritySkipped {
			t.Errorf("expected Skipped severity on missing command, got %v", finding.Severity)
		}
	})
}
