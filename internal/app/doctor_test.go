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
