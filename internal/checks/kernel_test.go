package checks

import (
	"errors"
	"testing"

	"faultradar/internal/config"
	"faultradar/internal/model"
	"faultradar/internal/system"
)

func TestCheckKernel(t *testing.T) {
	cfg := config.DefaultConfig()

	// 1. no errors -> ok
	t.Run("no errors -> ok", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte(""), nil
			},
		}
		finding := CheckKernel(runner, cfg)
		if finding.Severity != model.SeverityOK {
			t.Errorf("expected OK, got %v", finding.Severity)
		}
	})

	// 2. ACPI-only errors -> warning, not critical
	t.Run("ACPI-only errors -> warning", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("ACPI Error: AE_NOT_FOUND, While resolving a named reference package element\n"), nil
			},
		}
		finding := CheckKernel(runner, cfg)
		if finding.Severity != model.SeverityWarning {
			t.Errorf("expected Warning, got %v", finding.Severity)
		}
	})

	// 3. Snap blockdev errors -> warning, not critical
	t.Run("Snap blockdev errors -> warning", func(t *testing.T) {
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

	// 4. EXT4 error -> critical
	t.Run("EXT4 error -> critical", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("EXT4-fs error (device sda1): ext4_lookup: deleted inode referenced: 12345\n"), nil
			},
		}
		finding := CheckKernel(runner, cfg)
		if finding.Severity != model.SeverityCritical {
			t.Errorf("expected Critical, got %v", finding.Severity)
		}
	})

	// 5. NVMe timeout -> critical
	t.Run("NVMe timeout -> critical", func(t *testing.T) {
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

	// 6. OOM killer -> critical
	t.Run("OOM killer -> critical", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("Killed process 1234 (mysqld) total-vm:4300000kB, anon-rss:200000kB, file-rss:0kB, shmem-rss:0kB oom-killer\n"), nil
			},
		}
		finding := CheckKernel(runner, cfg)
		if finding.Severity != model.SeverityCritical {
			t.Errorf("expected Critical, got %v", finding.Severity)
		}
	})

	// 7. unknown low-count errors -> info
	t.Run("unknown low-count errors -> info", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("some unknown error 1\nsome unknown error 2\n"), nil
			},
		}
		finding := CheckKernel(runner, cfg)
		if finding.Severity != model.SeverityInfo {
			t.Errorf("expected Info, got %v", finding.Severity)
		}
	})

	// 8. unknown high-count errors -> warning
	t.Run("unknown high-count errors -> warning", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				lines := ""
				for i := 0; i < 15; i++ {
					lines += "some unknown error\n"
				}
				return []byte(lines), nil
			},
		}
		finding := CheckKernel(runner, cfg)
		if finding.Severity != model.SeverityWarning {
			t.Errorf("expected Warning, got %v", finding.Severity)
		}
	})

	// 9. command missing -> skipped
	t.Run("command missing -> skipped", func(t *testing.T) {
		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return nil, errors.New("exec: \"journalctl\": executable file not found in $PATH")
			},
		}
		finding := CheckKernel(runner, cfg)
		if finding.Severity != model.SeveritySkipped {
			t.Errorf("expected Skipped, got %v", finding.Severity)
		}
	})

	// 10. ignored pattern -> ignored from severity
	t.Run("ignored pattern -> ignored", func(t *testing.T) {
		customConfig := config.DefaultConfig()
		customConfig.Kernel.IgnorePatterns = []string{"ignore-me", "block.*dev"}

		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("ignore-me: ACPI Error\nCan't lookup blockdev\n"), nil
			},
		}
		finding := CheckKernel(runner, customConfig)
		// Should be OK because both lines are ignored
		if finding.Severity != model.SeverityOK {
			t.Errorf("expected OK for ignored errors, got %v", finding.Severity)
		}
	})

	// Additional test for downgrade pattern
	t.Run("downgrade pattern applied", func(t *testing.T) {
		customConfig := config.DefaultConfig()
		customConfig.Kernel.DowngradePatterns = []string{"sda1"}

		runner := system.FakeCommandRunner{
			RunFunc: func(name string, args ...string) ([]byte, error) {
				return []byte("EXT4-fs error (device sda1): error details\n"), nil
			},
		}
		finding := CheckKernel(runner, customConfig)
		// Normally EXT4-fs error is critical, but downgraded to warning
		if finding.Severity != model.SeverityWarning {
			t.Errorf("expected Warning for downgraded critical error, got %v", finding.Severity)
		}
	})
}
