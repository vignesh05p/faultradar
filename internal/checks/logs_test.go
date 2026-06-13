package checks

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"

	"faultradar/internal/model"
	"faultradar/internal/system"
)

func TestCheckLogs(t *testing.T) {
	config := model.DefaultConfig()

	t.Run("small /var/log", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			StatFunc: func(path string) (os.FileInfo, error) {
				return system.FakeFileInfo{NameVal: "log", IsDirVal: true}, nil
			},
			WalkDirFunc: func(root string, fn fs.WalkDirFunc) error {
				_ = fn("/var/log/syslog", system.FakeDirEntry{
					NameVal: "syslog",
					InfoVal: system.FakeFileInfo{NameVal: "syslog", SizeVal: 100 * 1024, IsDirVal: false},
				}, nil)
				return nil
			},
		}

		finding := CheckLogs(fsMock, config)
		if finding.Severity != model.SeverityOK {
			t.Errorf("expected OK severity, got %v", finding.Severity)
		}
	})

	t.Run("large /var/log", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			StatFunc: func(path string) (os.FileInfo, error) {
				return system.FakeFileInfo{NameVal: "log", IsDirVal: true}, nil
			},
			WalkDirFunc: func(root string, fn fs.WalkDirFunc) error {
				_ = fn("/var/log/syslog", system.FakeDirEntry{
					NameVal: "syslog",
					InfoVal: system.FakeFileInfo{NameVal: "syslog", SizeVal: 6 * 1024 * 1024 * 1024, IsDirVal: false},
				}, nil)
				return nil
			},
		}

		finding := CheckLogs(fsMock, config)
		if finding.Severity != model.SeverityCritical {
			t.Errorf("expected Critical severity, got %v", finding.Severity)
		}
	})

	t.Run("inaccessible /var/log", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			StatFunc: func(path string) (os.FileInfo, error) {
				return nil, errors.New("permission denied")
			},
		}

		finding := CheckLogs(fsMock, config)
		if finding.Severity != model.SeveritySkipped {
			t.Errorf("expected Skipped severity, got %v", finding.Severity)
		}
	})

	t.Run("includes biggest files", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			StatFunc: func(path string) (os.FileInfo, error) {
				return system.FakeFileInfo{NameVal: "log", IsDirVal: true}, nil
			},
			WalkDirFunc: func(root string, fn fs.WalkDirFunc) error {
				_ = fn("/var/log/syslog", system.FakeDirEntry{
					NameVal: "syslog",
					InfoVal: system.FakeFileInfo{NameVal: "syslog", SizeVal: 100 * 1024 * 1024, IsDirVal: false},
				}, nil)
				_ = fn("/var/log/auth.log", system.FakeDirEntry{
					NameVal: "auth.log",
					InfoVal: system.FakeFileInfo{NameVal: "auth.log", SizeVal: 200 * 1024 * 1024, IsDirVal: false},
				}, nil)
				return nil
			},
		}

		finding := CheckLogs(fsMock, config)
		foundAuth := false
		foundSyslog := false
		for _, detail := range finding.Details {
			if strings.Contains(detail, "auth.log") {
				foundAuth = true
			}
			if strings.Contains(detail, "syslog") {
				foundSyslog = true
			}
		}
		if !foundAuth || !foundSyslog {
			t.Errorf("expected details to contain syslog and auth.log info")
		}
	})
}
