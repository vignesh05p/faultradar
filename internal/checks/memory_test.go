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

	t.Run("normal memory", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			ReadFileFunc: func(path string) ([]byte, error) {
				return []byte("MemTotal:       10000 kB\nMemAvailable:    8000 kB\nSwapTotal:       2000 kB\nSwapFree:        2000 kB\n"), nil
			},
		}

		findings := CheckMemory(fsMock, cfg)
		if len(findings) != 1 {
			t.Fatalf("expected 1 finding, got %d", len(findings))
		}
		if findings[0].ID != "memory.ok" {
			t.Errorf("expected memory.ok, got %s", findings[0].ID)
		}
		if findings[0].Severity != model.SeverityOK {
			t.Errorf("expected OK severity, got %v", findings[0].Severity)
		}
	})

	t.Run("low memory warning", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			ReadFileFunc: func(path string) ([]byte, error) {
				return []byte("MemTotal:       10000 kB\nMemAvailable:     800 kB\nSwapTotal:       2000 kB\nSwapFree:        2000 kB\n"), nil
			},
		}

		findings := CheckMemory(fsMock, cfg)
		if len(findings) != 1 {
			t.Fatalf("expected 1 finding, got %d", len(findings))
		}
		if findings[0].ID != "memory.low" {
			t.Errorf("expected memory.low, got %s", findings[0].ID)
		}
		if findings[0].Severity != model.SeverityWarning {
			t.Errorf("expected Warning severity, got %v", findings[0].Severity)
		}
	})

	t.Run("low memory critical", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			ReadFileFunc: func(path string) ([]byte, error) {
				return []byte("MemTotal:       10000 kB\nMemAvailable:     400 kB\nSwapTotal:       2000 kB\nSwapFree:        2000 kB\n"), nil
			},
		}

		findings := CheckMemory(fsMock, cfg)
		if len(findings) != 1 {
			t.Fatalf("expected 1 finding, got %d", len(findings))
		}
		if findings[0].ID != "memory.critical" {
			t.Errorf("expected memory.critical, got %s", findings[0].ID)
		}
		if findings[0].Severity != model.SeverityCritical {
			t.Errorf("expected Critical severity, got %v", findings[0].Severity)
		}
	})

	t.Run("no swap warning", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			ReadFileFunc: func(path string) ([]byte, error) {
				return []byte("MemTotal:       10000 kB\nMemAvailable:    8000 kB\nSwapTotal:          0 kB\nSwapFree:           0 kB\n"), nil
			},
		}

		findings := CheckMemory(fsMock, cfg)
		if len(findings) != 1 {
			t.Fatalf("expected 1 finding, got %d", len(findings))
		}
		if findings[0].ID != "memory.no_swap" {
			t.Errorf("expected memory.no_swap, got %s", findings[0].ID)
		}
		if findings[0].Severity != model.SeverityWarning {
			t.Errorf("expected Warning severity on no swap, got %v", findings[0].Severity)
		}
	})

	t.Run("both low memory and no swap", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			ReadFileFunc: func(path string) ([]byte, error) {
				return []byte("MemTotal:       10000 kB\nMemAvailable:     800 kB\nSwapTotal:          0 kB\nSwapFree:           0 kB\n"), nil
			},
		}

		findings := CheckMemory(fsMock, cfg)
		if len(findings) != 2 {
			t.Fatalf("expected 2 findings, got %d", len(findings))
		}
	})

	t.Run("malformed meminfo", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			ReadFileFunc: func(path string) ([]byte, error) {
				return []byte("garbage content\n"), nil
			},
		}

		findings := CheckMemory(fsMock, cfg)
		if len(findings) != 1 || findings[0].ID != "memory.unavailable" {
			t.Errorf("expected memory.unavailable on malformed file, got %+v", findings)
		}
	})

	t.Run("missing meminfo", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			ReadFileFunc: func(path string) ([]byte, error) {
				return nil, errors.New("file not found")
			},
		}

		findings := CheckMemory(fsMock, cfg)
		if len(findings) != 1 || findings[0].Severity != model.SeveritySkipped {
			t.Errorf("expected Skipped severity on missing file, got %+v", findings)
		}
	})
}
