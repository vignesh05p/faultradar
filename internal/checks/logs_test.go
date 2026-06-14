package checks

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"

	"faultradar/internal/config"
	"faultradar/internal/model"
	"faultradar/internal/system"
)

func TestCheckLogs(t *testing.T) {
	cfg := config.DefaultConfig()

	t.Run("normal log usage", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			StatFunc: func(path string) (os.FileInfo, error) {
				return system.FakeFileInfo{NameVal: "log", IsDirVal: true}, nil
			},
			WalkDirFunc: func(root string, fn fs.WalkDirFunc) error {
				_ = fn("/var/log/syslog", system.FakeDirEntry{
					NameVal: "syslog",
					InfoVal: system.FakeFileInfo{NameVal: "syslog", SizeVal: 10 * 1024 * 1024, IsDirVal: false},
				}, nil)
				return nil
			},
		}
		finding := CheckLogs(fsMock, cfg)
		if finding.Severity != model.SeverityOK {
			t.Errorf("expected OK, got %v", finding.Severity)
		}
	})

	t.Run("warning actual disk usage", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			StatFunc: func(path string) (os.FileInfo, error) {
				return system.FakeFileInfo{NameVal: "log", IsDirVal: true}, nil
			},
			WalkDirFunc: func(root string, fn fs.WalkDirFunc) error {
				_ = fn("/var/log/syslog", system.FakeDirEntry{
					NameVal: "syslog",
					InfoVal: system.FakeFileInfo{NameVal: "syslog", SizeVal: 1500 * 1024 * 1024, ActualSizeVal: 1500 * 1024 * 1024, IsDirVal: false},
				}, nil)
				return nil
			},
		}
		finding := CheckLogs(fsMock, cfg)
		if finding.Severity != model.SeverityWarning {
			t.Errorf("expected Warning, got %v", finding.Severity)
		}
		if !strings.Contains(finding.Suggestion, "Inspect the largest logs and fix the source before deleting") {
			t.Errorf("expected suggestion to advise source fixes, got: %s", finding.Suggestion)
		}
	})

	t.Run("critical actual disk usage", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			StatFunc: func(path string) (os.FileInfo, error) {
				return system.FakeFileInfo{NameVal: "log", IsDirVal: true}, nil
			},
			WalkDirFunc: func(root string, fn fs.WalkDirFunc) error {
				_ = fn("/var/log/syslog", system.FakeDirEntry{
					NameVal: "syslog",
					InfoVal: system.FakeFileInfo{NameVal: "syslog", SizeVal: 6 * 1024 * 1024 * 1024, ActualSizeVal: 6 * 1024 * 1024 * 1024, IsDirVal: false},
				}, nil)
				return nil
			},
		}
		finding := CheckLogs(fsMock, cfg)
		if finding.Severity != model.SeverityCritical {
			t.Errorf("expected Critical, got %v", finding.Severity)
		}
	})

	t.Run("missing /var/log handled gracefully", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			StatFunc: func(path string) (os.FileInfo, error) {
				return nil, errors.New("permission denied")
			},
		}
		finding := CheckLogs(fsMock, cfg)
		if finding.Severity != model.SeveritySkipped {
			t.Errorf("expected Skipped, got %v", finding.Severity)
		}
	})

	t.Run("largest files formatting", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			StatFunc: func(path string) (os.FileInfo, error) {
				return system.FakeFileInfo{NameVal: "log", IsDirVal: true}, nil
			},
			WalkDirFunc: func(root string, fn fs.WalkDirFunc) error {
				_ = fn("/var/log/syslog", system.FakeDirEntry{
					NameVal: "syslog",
					InfoVal: system.FakeFileInfo{NameVal: "syslog", SizeVal: 100 * 1024 * 1024, IsDirVal: false},
				}, nil)
				_ = fn("/var/log/nginx/access.log", system.FakeDirEntry{
					NameVal: "access.log",
					InfoVal: system.FakeFileInfo{NameVal: "access.log", SizeVal: 50 * 1024 * 1024, IsDirVal: false},
				}, nil)
				return nil
			},
		}
		finding := CheckLogs(fsMock, cfg)
		var hasSyslog, hasNginx bool
		for _, detail := range finding.Details {
			if strings.Contains(detail, "/var/log/syslog: 100.00 MB") {
				hasSyslog = true
			}
			if strings.Contains(detail, "/var/log/nginx/access.log: 50.00 MB") {
				hasNginx = true
			}
		}
		if !hasSyslog || !hasNginx {
			t.Errorf("expected largest files to list syslog and nginx access.log")
		}
	})

	t.Run("largest directories formatting", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			StatFunc: func(path string) (os.FileInfo, error) {
				return system.FakeFileInfo{NameVal: "log", IsDirVal: true}, nil
			},
			WalkDirFunc: func(root string, fn fs.WalkDirFunc) error {
				_ = fn("/var/log/journal/123/system.journal", system.FakeDirEntry{
					NameVal: "system.journal",
					InfoVal: system.FakeFileInfo{NameVal: "system.journal", SizeVal: 200 * 1024 * 1024, IsDirVal: false},
				}, nil)
				_ = fn("/var/log/mongodb/mongod.log", system.FakeDirEntry{
					NameVal: "mongod.log",
					InfoVal: system.FakeFileInfo{NameVal: "mongod.log", SizeVal: 50 * 1024 * 1024, IsDirVal: false},
				}, nil)
				return nil
			},
		}
		finding := CheckLogs(fsMock, cfg)
		var hasJournalDir, hasMongoDir bool
		for _, detail := range finding.Details {
			if strings.Contains(detail, "/var/log/journal: 200.00 MB") {
				hasJournalDir = true
			}
			if strings.Contains(detail, "/var/log/mongodb: 50.00 MB") {
				hasMongoDir = true
			}
		}
		if !hasJournalDir || !hasMongoDir {
			t.Errorf("expected largest directories to list aggregated sizes for journal and mongodb")
		}
	})

	t.Run("sparse files detected", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			StatFunc: func(path string) (os.FileInfo, error) {
				return system.FakeFileInfo{NameVal: "log", IsDirVal: true}, nil
			},
			WalkDirFunc: func(root string, fn fs.WalkDirFunc) error {
				_ = fn("/var/log/lastlog", system.FakeDirEntry{
					NameVal: "lastlog",
					InfoVal: system.FakeFileInfo{NameVal: "lastlog", SizeVal: 500 * 1024 * 1024, ActualSizeVal: 1024, IsDirVal: false},
				}, nil)
				return nil
			},
		}
		finding := CheckLogs(fsMock, cfg)
		var hasSparseNote bool
		for _, detail := range finding.Details {
			if strings.Contains(detail, "/var/log/lastlog appears sparse") {
				hasSparseNote = true
			}
		}
		if !hasSparseNote {
			t.Errorf("expected details to include sparse file notice")
		}
	})

	t.Run("sparse files do not cause false warning or critical", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			StatFunc: func(path string) (os.FileInfo, error) {
				return system.FakeFileInfo{NameVal: "log", IsDirVal: true}, nil
			},
			WalkDirFunc: func(root string, fn fs.WalkDirFunc) error {
				_ = fn("/var/log/lastlog", system.FakeDirEntry{
					NameVal: "lastlog",
					InfoVal: system.FakeFileInfo{NameVal: "lastlog", SizeVal: 6 * 1024 * 1024 * 1024, ActualSizeVal: 10 * 1024 * 1024, IsDirVal: false},
				}, nil)
				return nil
			},
		}
		finding := CheckLogs(fsMock, cfg)
		if finding.Severity != model.SeverityOK {
			t.Errorf("expected OK severity due to small actual size, got %v", finding.Severity)
		}
	})

	t.Run("permission errors do not crash", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			StatFunc: func(path string) (os.FileInfo, error) {
				return system.FakeFileInfo{NameVal: "log", IsDirVal: true}, nil
			},
			WalkDirFunc: func(root string, fn fs.WalkDirFunc) error {
				_ = fn("/var/log/syslog", system.FakeDirEntry{
					NameVal: "syslog",
					InfoVal: system.FakeFileInfo{NameVal: "syslog", SizeVal: 1024, IsDirVal: false},
				}, errors.New("permission denied"))
				return nil
			},
		}
		finding := CheckLogs(fsMock, cfg)
		if finding.Severity != model.SeverityOK {
			t.Errorf("expected OK when walk errors are skipped, got %v", finding.Severity)
		}
	})
}
