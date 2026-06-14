package checks

import (
	"errors"
	"strings"
	"testing"

	"faultradar/internal/config"
	"faultradar/internal/model"
	"faultradar/internal/system"
)

func TestCheckKernel(t *testing.T) {
	cfg := config.DefaultConfig()

	t.Run("no kernel errors", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte(""), nil
			},
		}
		finding := CheckKernel(runner, cfg)
		if finding.ID != "kernel.errors.none" {
			t.Errorf("expected kernel.errors.none, got %s", finding.ID)
		}
		if finding.Severity != model.SeverityOK {
			t.Errorf("expected OK, got %v", finding.Severity)
		}
	})

	t.Run("ACPI warning pattern", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("ACPI Error: AE_NOT_FOUND, While resolving a named reference package element\n"), nil
			},
		}
		finding := CheckKernel(runner, cfg)
		if finding.ID != "kernel.errors.warning" {
			t.Errorf("expected kernel.errors.warning, got %s", finding.ID)
		}
		if finding.Severity != model.SeverityWarning {
			t.Errorf("expected Warning, got %v", finding.Severity)
		}
	})

	t.Run("Snap blockdev warning pattern", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("/var/lib/snapd/snaps/chromium_3235.snap: Can't lookup blockdev\n"), nil
			},
		}
		finding := CheckKernel(runner, cfg)
		if finding.Severity != model.SeverityWarning {
			t.Errorf("expected Warning, got %v", finding.Severity)
		}
	})

	t.Run("EXT4 error critical", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("EXT4-fs error (device sda1): ext4_lookup: deleted inode referenced: 12345\n"), nil
			},
		}
		finding := CheckKernel(runner, cfg)
		if finding.ID != "kernel.errors.critical" {
			t.Errorf("expected kernel.errors.critical, got %s", finding.ID)
		}
		if finding.Severity != model.SeverityCritical {
			t.Errorf("expected Critical, got %v", finding.Severity)
		}
	})

	t.Run("NVMe timeout critical", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("nvme nvme0: Device not ready; aborting initialisation, nvme timeout\n"), nil
			},
		}
		finding := CheckKernel(runner, cfg)
		if finding.Severity != model.SeverityCritical {
			t.Errorf("expected Critical, got %v", finding.Severity)
		}
	})

	t.Run("OOM killer critical", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("Killed process 1234 (mysqld) total-vm:4300000kB oom-killer\n"), nil
			},
		}
		finding := CheckKernel(runner, cfg)
		if finding.Severity != model.SeverityCritical {
			t.Errorf("expected Critical, got %v", finding.Severity)
		}
	})

	t.Run("I/O error critical", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("Buffer I/O error on device sda1, logical block 12345\n"), nil
			},
		}
		finding := CheckKernel(runner, cfg)
		if finding.Severity != model.SeverityCritical {
			t.Errorf("expected Critical, got %v", finding.Severity)
		}
	})

	t.Run("journalctl unavailable", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return nil, errors.New("exec: \"journalctl\": executable file not found in $PATH")
			},
		}
		finding := CheckKernel(runner, cfg)
		if finding.ID != "kernel.unavailable" {
			t.Errorf("expected kernel.unavailable, got %s", finding.ID)
		}
		if finding.Severity != model.SeveritySkipped {
			t.Errorf("expected Skipped, got %v", finding.Severity)
		}
	})

	t.Run("restricted permission output", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("Hint: You are currently not seeing messages from other users and the system.\n"), errors.New("exit status 1: permission denied")
			},
		}
		finding := CheckKernel(runner, cfg)
		if finding.ID != "kernel.restricted" {
			t.Errorf("expected kernel.restricted, got %s", finding.ID)
		}
		if finding.Severity != model.SeveritySkipped {
			t.Errorf("expected Skipped, got %v", finding.Severity)
		}
	})

	t.Run("ignore patterns", func(t *testing.T) {
		customConfig := config.DefaultConfig()
		customConfig.Kernel.IgnorePatterns = []string{"ignore-me", "block.*dev"}

		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("ignore-me: ACPI Error\nCan't lookup blockdev\n"), nil
			},
		}
		finding := CheckKernel(runner, customConfig)
		if finding.Severity != model.SeverityOK {
			t.Errorf("expected OK for ignored errors, got %v", finding.Severity)
		}
	})

	t.Run("downgrade patterns", func(t *testing.T) {
		customConfig := config.DefaultConfig()
		customConfig.Kernel.DowngradePatterns = []string{"sda1"}

		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("EXT4-fs error (device sda1): error details\n"), nil
			},
		}
		finding := CheckKernel(runner, customConfig)
		if finding.Severity != model.SeverityWarning {
			t.Errorf("expected Warning for downgraded critical error, got %v", finding.Severity)
		}
	})

	t.Run("long examples truncated", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				lines := ""
				for i := 0; i < 10; i++ {
					lines += "ACPI Error: AE_NOT_FOUND line " + strings.Repeat("x", 200) + "\n"
				}
				return []byte(lines), nil
			},
		}
		finding := CheckKernel(runner, cfg)
		var exampleCount int
		for _, detail := range finding.Details {
			if strings.HasPrefix(detail, "  - ") {
				exampleCount++
			}
		}
		if exampleCount > 4 {
			t.Errorf("expected truncated examples, got %d example lines", exampleCount)
		}
	})
}

func TestValidateKernelPatterns(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Kernel.IgnorePatterns = []string{"valid", "[invalid"}
	cfg.Kernel.DowngradePatterns = []string{"also[bad"}

	findings := ValidateKernelPatterns(cfg)
	if len(findings) != 2 {
		t.Fatalf("expected 2 invalid pattern findings, got %d", len(findings))
	}
	for _, f := range findings {
		if f.Severity != model.SeverityWarning {
			t.Errorf("expected warning for invalid pattern, got %v", f.Severity)
		}
	}
}
