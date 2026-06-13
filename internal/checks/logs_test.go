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

	// 1. small logs -> ok
	t.Run("small logs -> ok", func(t *testing.T) {
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
		finding := CheckLogs(fsMock, config)
		if finding.Severity != model.SeverityOK {
			t.Errorf("expected OK, got %v", finding.Severity)
		}
	})

	// 2. warning threshold -> warning
	t.Run("warning threshold -> warning", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			StatFunc: func(path string) (os.FileInfo, error) {
				return system.FakeFileInfo{NameVal: "log", IsDirVal: true}, nil
			},
			WalkDirFunc: func(root string, fn fs.WalkDirFunc) error {
				// 1.5 GB is >= 1024 MB warning threshold
				_ = fn("/var/log/syslog", system.FakeDirEntry{
					NameVal: "syslog",
					InfoVal: system.FakeFileInfo{NameVal: "syslog", SizeVal: 1500 * 1024 * 1024, IsDirVal: false},
				}, nil)
				return nil
			},
		}
		finding := CheckLogs(fsMock, config)
		if finding.Severity != model.SeverityWarning {
			t.Errorf("expected Warning, got %v", finding.Severity)
		}
		if !strings.Contains(finding.Suggestion, "Inspect the largest logs and fix the source before deleting") {
			t.Errorf("expected suggestion to advise source fixes, got: %s", finding.Suggestion)
		}
	})

	// 3. critical threshold -> critical
	t.Run("critical threshold -> critical", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			StatFunc: func(path string) (os.FileInfo, error) {
				return system.FakeFileInfo{NameVal: "log", IsDirVal: true}, nil
			},
			WalkDirFunc: func(root string, fn fs.WalkDirFunc) error {
				// 6 GB is >= 5120 MB critical threshold
				_ = fn("/var/log/syslog", system.FakeDirEntry{
					NameVal: "syslog",
					InfoVal: system.FakeFileInfo{NameVal: "syslog", SizeVal: 6 * 1024 * 1024 * 1024, IsDirVal: false},
				}, nil)
				return nil
			},
		}
		finding := CheckLogs(fsMock, config)
		if finding.Severity != model.SeverityCritical {
			t.Errorf("expected Critical, got %v", finding.Severity)
		}
	})

	// 4. inaccessible /var/log -> skipped
	t.Run("inaccessible /var/log -> skipped", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			StatFunc: func(path string) (os.FileInfo, error) {
				return nil, errors.New("permission denied")
			},
		}
		finding := CheckLogs(fsMock, config)
		if finding.Severity != model.SeveritySkipped {
			t.Errorf("expected Skipped, got %v", finding.Severity)
		}
	})

	// 5. largest files included
	t.Run("largest files included", func(t *testing.T) {
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
		finding := CheckLogs(fsMock, config)
		var hasSyslog, hasNginx bool
		for _, detail := range finding.Details {
			if strings.Contains(detail, "/var/log/syslog:") && strings.Contains(detail, "100.00 MB") {
				hasSyslog = true
			}
			if strings.Contains(detail, "/var/log/nginx/access.log:") && strings.Contains(detail, "50.00 MB") {
				hasNginx = true
			}
		}
		if !hasSyslog || !hasNginx {
			t.Errorf("expected largest files to list syslog and nginx access.log")
		}
	})

	// 6. largest directories included
	t.Run("largest directories included", func(t *testing.T) {
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
		finding := CheckLogs(fsMock, config)
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

	// 7. sparse file note included if lastlog/btmp/wtmp appears
	t.Run("sparse file note included if lastlog/btmp/wtmp appears", func(t *testing.T) {
		fsMock := system.FakeFileSystem{
			StatFunc: func(path string) (os.FileInfo, error) {
				return system.FakeFileInfo{NameVal: "log", IsDirVal: true}, nil
			},
			WalkDirFunc: func(root string, fn fs.WalkDirFunc) error {
				_ = fn("/var/log/lastlog", system.FakeDirEntry{
					NameVal: "lastlog",
					InfoVal: system.FakeFileInfo{NameVal: "lastlog", SizeVal: 500 * 1024 * 1024, IsDirVal: false},
				}, nil)
				return nil
			},
		}
		finding := CheckLogs(fsMock, config)
		var hasSparseNote bool
		for _, detail := range finding.Details {
			if strings.Contains(detail, "lastlog, btmp, and wtmp may be sparse or misleading") {
				hasSparseNote = true
			}
		}
		if !hasSparseNote {
			t.Errorf("expected details to include sparse file notice")
		}
	})
}
