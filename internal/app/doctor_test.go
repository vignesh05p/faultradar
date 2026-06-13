package app

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
	"testing"

	"faultradar/internal/model"
	"faultradar/internal/system"
)

func TestDoctorRun(t *testing.T) {
	config := model.DefaultConfig()

	fsMock := system.FakeFileSystem{
		StatfsFunc: func(path string) (*syscall.Statfs_t, error) {
			return &syscall.Statfs_t{Blocks: 100, Bfree: 80, Bavail: 80}, nil
		},
		StatFunc: func(path string) (os.FileInfo, error) {
			return system.FakeFileInfo{NameVal: "log", IsDirVal: true}, nil
		},
		WalkDirFunc: func(root string, fn fs.WalkDirFunc) error {
			return nil
		},
		ReadFileFunc: func(path string) ([]byte, error) {
			return []byte("MemTotal:       10000 kB\nMemAvailable:    8000 kB\nSwapTotal:       2000 kB\nSwapFree:        2000 kB\n"), nil
		},
	}

	runnerMock := system.FakeCommandRunner{
		RunFunc: func(name string, args ...string) ([]byte, error) {
			if name == "systemctl" {
				return []byte("0 loaded units listed.\n"), nil
			}
			if name == "journalctl" {
				return []byte(""), nil
			}
			return nil, errors.New("unknown command")
		},
	}

	doc := Doctor{
		Config: config,
		Runner: runnerMock,
		FS:     fsMock,
	}

	findings := doc.Run()

	expectedIDs := map[string]bool{
		"disk.root.usage":            false,
		"logs.varlog.size":           false,
		"systemd.failed_units":       false,
		"kernel.errors.current_boot": false,
		"memory.available":           false,
	}

	if len(findings) != len(expectedIDs) {
		t.Errorf("expected %d findings, got %d", len(expectedIDs), len(findings))
	}

	for _, f := range findings {
		if _, ok := expectedIDs[f.ID]; ok {
			expectedIDs[f.ID] = true
		} else {
			t.Errorf("unexpected finding ID: %s", f.ID)
		}
	}

	for id, found := range expectedIDs {
		if !found {
			t.Errorf("expected finding ID %s was not returned", id)
		}
	}
}

func TestLoadConfig(t *testing.T) {
	// 1. defaults load
	t.Run("defaults load when no file exists", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			ReadFileFunc: func(path string) ([]byte, error) {
				return nil, fs.ErrNotExist
			},
		}

		config, findings := LoadConfig(fsMock)
		if len(findings) != 0 {
			t.Errorf("expected no config findings, got %d", len(findings))
		}
		if config.Disk.RootWarningPercent != 85 {
			t.Errorf("expected default root warning percent 85, got %d", config.Disk.RootWarningPercent)
		}
	})

	// 2. user config overrides defaults
	t.Run("user config overrides defaults", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			ReadFileFunc: func(path string) ([]byte, error) {
				// Supplying custom warning percent for disk and ignore units for systemd
				return []byte(`{
					"disk": {
						"root_warning_percent": 90
					},
					"systemd": {
						"ignore_units": ["cups.service"]
					}
				}`), nil
			},
		}

		config, findings := LoadConfig(fsMock)
		if len(findings) != 0 {
			t.Errorf("expected no config findings, got %d", len(findings))
		}
		if config.Disk.RootWarningPercent != 90 {
			t.Errorf("expected overridden disk root warning percent 90, got %d", config.Disk.RootWarningPercent)
		}
		if config.Disk.RootCriticalPercent != 95 {
			t.Errorf("expected default disk root critical percent 95 to remain, got %d", config.Disk.RootCriticalPercent)
		}
		if len(config.Systemd.IgnoreUnits) != 1 || config.Systemd.IgnoreUnits[0] != "cups.service" {
			t.Errorf("expected systemd ignore_units overridden, got %v", config.Systemd.IgnoreUnits)
		}
	})

	// 3. invalid config handled cleanly
	t.Run("invalid config handled cleanly", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			ReadFileFunc: func(path string) ([]byte, error) {
				return []byte(`{invalid-json}`), nil
			},
		}

		config, findings := LoadConfig(fsMock)
		if len(findings) != 1 {
			t.Errorf("expected 1 finding for config load error, got %d", len(findings))
		}
		if findings[0].Severity != model.SeverityWarning {
			t.Errorf("expected warning severity, got %v", findings[0].Severity)
		}
		if config.Disk.RootWarningPercent != 85 {
			t.Errorf("expected default fallback config on load error, got %d", config.Disk.RootWarningPercent)
		}
	})
}
