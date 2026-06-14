package checks

import (
	"errors"
	"testing"

	"faultradar/internal/config"
	"faultradar/internal/model"
	"faultradar/internal/system"
)

func TestCheckMemory(t *testing.T) {
	cfg := config.DefaultConfig()

	t.Run("healthy meminfo", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			ReadFileFunc: func(path string) ([]byte, error) {
				return []byte("MemTotal:       10000 kB\nMemAvailable:    8000 kB\nSwapTotal:       2000 kB\nSwapFree:        2000 kB\n"), nil
			},
		}

		finding := CheckMemory(fsMock, cfg)
		if finding.Severity != model.SeverityOK {
			t.Errorf("expected OK severity, got %v", finding.Severity)
		}
	})

	t.Run("low memory warning", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			ReadFileFunc: func(path string) ([]byte, error) {
				return []byte("MemTotal:       10000 kB\nMemAvailable:    1200 kB\nSwapTotal:       2000 kB\nSwapFree:        2000 kB\n"), nil
			},
		}

		finding := CheckMemory(fsMock, cfg)
		if finding.Severity != model.SeverityWarning {
			t.Errorf("expected Warning severity, got %v", finding.Severity)
		}
	})

	t.Run("critical memory", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			ReadFileFunc: func(path string) ([]byte, error) {
				return []byte("MemTotal:       10000 kB\nMemAvailable:     400 kB\nSwapTotal:       2000 kB\nSwapFree:        2000 kB\n"), nil
			},
		}

		finding := CheckMemory(fsMock, cfg)
		if finding.Severity != model.SeverityCritical {
			t.Errorf("expected Critical severity, got %v", finding.Severity)
		}
	})

	t.Run("no swap warning", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			ReadFileFunc: func(path string) ([]byte, error) {
				return []byte("MemTotal:       10000 kB\nMemAvailable:    2000 kB\nSwapTotal:          0 kB\nSwapFree:           0 kB\n"), nil
			},
		}

		finding := CheckMemory(fsMock, cfg)
		if finding.Severity != model.SeverityWarning {
			t.Errorf("expected Warning severity on no swap, got %v", finding.Severity)
		}
	})

	t.Run("malformed meminfo", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			ReadFileFunc: func(path string) ([]byte, error) {
				return []byte("garbage content\n"), nil
			},
		}

		finding := CheckMemory(fsMock, cfg)
		if finding.Severity != model.SeveritySkipped {
			t.Errorf("expected Skipped severity on malformed file, got %v", finding.Severity)
		}
	})

	t.Run("missing meminfo", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			ReadFileFunc: func(path string) ([]byte, error) {
				return nil, errors.New("file not found")
			},
		}

		finding := CheckMemory(fsMock, cfg)
		if finding.Severity != model.SeveritySkipped {
			t.Errorf("expected Skipped severity on missing file, got %v", finding.Severity)
		}
	})
}
