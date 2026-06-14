package checks

import (
	"errors"
	"testing"

	"faultradar/internal/config"
	"faultradar/internal/model"
	"faultradar/internal/system"
)

func TestCheckDisk(t *testing.T) {
	cfg := config.DefaultConfig()

	t.Run("normal usage", func(t *testing.T) {
		fs := system.FakeFileSystem{
			StatfsFunc: func(path string) (system.FsInfo, error) {
				return system.FsInfo{
					Blocks: 100,
					Bfree:  50,
					Bavail: 50,
				}, nil
			},
		}

		finding := CheckDisk(fs, cfg)
		if finding.Severity != model.SeverityOK {
			t.Errorf("expected OK severity, got %v", finding.Severity)
		}
		if finding.ID != "disk.root.usage" {
			t.Errorf("expected ID disk.root.usage, got %v", finding.ID)
		}
	})

	t.Run("warning usage", func(t *testing.T) {
		fs := system.FakeFileSystem{
			StatfsFunc: func(path string) (system.FsInfo, error) {
				return system.FsInfo{
					Blocks: 100,
					Bfree:  15,
					Bavail: 15,
				}, nil
			},
		}

		finding := CheckDisk(fs, cfg)
		if finding.Severity != model.SeverityWarning {
			t.Errorf("expected Warning severity, got %v", finding.Severity)
		}
	})

	t.Run("critical usage", func(t *testing.T) {
		fs := system.FakeFileSystem{
			StatfsFunc: func(path string) (system.FsInfo, error) {
				return system.FsInfo{
					Blocks: 100,
					Bfree:  4,
					Bavail: 4,
				}, nil
			},
		}

		finding := CheckDisk(fs, cfg)
		if finding.Severity != model.SeverityCritical {
			t.Errorf("expected Critical severity, got %v", finding.Severity)
		}
	})

	t.Run("statfs error", func(t *testing.T) {
		fs := system.FakeFileSystem{
			StatfsFunc: func(path string) (system.FsInfo, error) {
				return system.FsInfo{}, errors.New("statfs failed")
			},
		}

		finding := CheckDisk(fs, cfg)
		if finding.Severity != model.SeveritySkipped {
			t.Errorf("expected Skipped severity on error, got %v", finding.Severity)
		}
	})
}
