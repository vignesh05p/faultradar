package app

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	"faultradar/internal/config"
	"faultradar/internal/model"
	"faultradar/internal/system"
)

func TestDoctorRun(t *testing.T) {
	cfg := config.DefaultConfig()

	fsMock := system.FakeFileSystem{
		StatfsFunc: func(path string) (system.FsInfo, error) {
			return system.FsInfo{Blocks: 100, Bfree: 80, Bavail: 80}, nil
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
		Config: cfg,
		Runner: runnerMock,
		FS:     fsMock,
	}

	findings := doc.Run()

	requiredIDs := []string{
		"disk.root.usage",
		"logs.varlog.size",
		"systemd.failed.none",
		"kernel.errors.none",
		"memory.ok",
	}

	if len(findings) < len(requiredIDs) {
		t.Errorf("expected at least %d findings, got %d", len(requiredIDs), len(findings))
	}

	found := make(map[string]bool)
	for _, f := range findings {
		found[f.ID] = true
	}
	for _, id := range requiredIDs {
		if !found[id] {
			t.Errorf("expected finding ID %s was not returned", id)
		}
	}
}

func TestExitCode(t *testing.T) {
	t.Run("exit code 2 for critical", func(t *testing.T) {
		code := ExitCode([]model.Finding{{Severity: model.SeverityCritical}})
		if code != 2 {
			t.Errorf("expected exit code 2, got %d", code)
		}
	})

	t.Run("exit code 1 for warning", func(t *testing.T) {
		code := ExitCode([]model.Finding{{Severity: model.SeverityWarning}})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})

	t.Run("exit code 0 for only ok/skipped/info", func(t *testing.T) {
		code := ExitCode([]model.Finding{
			{Severity: model.SeverityOK},
			{Severity: model.SeveritySkipped},
			{Severity: model.SeverityInfo},
		})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	t.Run("critical beats warning", func(t *testing.T) {
		code := ExitCode([]model.Finding{
			{Severity: model.SeverityWarning},
			{Severity: model.SeverityCritical},
		})
		if code != 2 {
			t.Errorf("expected exit code 2, got %d", code)
		}
	})
}

func TestLoadConfig(t *testing.T) {
	t.Run("defaults load when no file exists", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			ReadFileFunc: func(path string) ([]byte, error) {
				return nil, fs.ErrNotExist
			},
		}

		loadedCfg, findings := LoadConfig(fsMock)
		if len(findings) != 0 {
			t.Errorf("expected no config findings, got %d", len(findings))
		}
		if loadedCfg.Disk.WarningPercent != 90 {
			t.Errorf("expected default disk warning percent 90, got %d", loadedCfg.Disk.WarningPercent)
		}
		if loadedCfg.Disk.CriticalPercent != 97 {
			t.Errorf("expected default disk critical percent 97, got %d", loadedCfg.Disk.CriticalPercent)
		}
	})

	t.Run("custom thresholds load", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			ReadFileFunc: func(path string) ([]byte, error) {
				return []byte(`{
					"disk": {
						"warning_percent": 80,
						"critical_percent": 95
					},
					"logs": {
						"warning_mb": 512,
						"critical_mb": 2048
					},
					"memory": {
						"warning_available_percent": 20,
						"critical_available_percent": 8
					}
				}`), nil
			},
		}

		loadedCfg, findings := LoadConfig(fsMock)
		if len(findings) != 0 {
			t.Errorf("expected no config findings, got %d", len(findings))
		}
		if loadedCfg.Disk.WarningPercent != 80 {
			t.Errorf("expected disk warning percent 80, got %d", loadedCfg.Disk.WarningPercent)
		}
		if loadedCfg.Logs.WarningMB != 512 {
			t.Errorf("expected logs warning_mb 512, got %d", loadedCfg.Logs.WarningMB)
		}
		if loadedCfg.Memory.WarningAvailablePercent != 20 {
			t.Errorf("expected memory warning_available_percent 20, got %d", loadedCfg.Memory.WarningAvailablePercent)
		}
	})

	t.Run("invalid config handled safely", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			ReadFileFunc: func(path string) ([]byte, error) {
				return []byte(`{invalid-json}`), nil
			},
		}

		loadedCfg, findings := LoadConfig(fsMock)
		if len(findings) != 1 {
			t.Errorf("expected 1 finding for config load error, got %d", len(findings))
		}
		if findings[0].Severity != model.SeverityWarning {
			t.Errorf("expected warning severity, got %v", findings[0].Severity)
		}
		if loadedCfg.Disk.WarningPercent != 90 {
			t.Errorf("expected default fallback config on load error, got %d", loadedCfg.Disk.WarningPercent)
		}
	})

	t.Run("invalid regex handled safely", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			ReadFileFunc: func(path string) ([]byte, error) {
				return []byte(`{
					"kernel": {
						"ignore_patterns": ["[bad"]
					}
				}`), nil
			},
		}

		_, findings := LoadConfig(fsMock)
		if len(findings) != 1 {
			t.Errorf("expected 1 finding for invalid regex, got %d", len(findings))
		}
		if findings[0].Severity != model.SeverityWarning {
			t.Errorf("expected warning severity for invalid regex, got %v", findings[0].Severity)
		}
	})
}
