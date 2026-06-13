package checks

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"faultradar/internal/model"
	"faultradar/internal/system"
)

type fileInfo struct {
	Path string
	Size int64
}

type dirInfo struct {
	Path string
	Size int64
}

func getVarLogSubdir(path string) string {
	const prefix = "/var/log"
	if path == prefix {
		return prefix
	}
	// Extract relative path after /var/log
	rel, err := filepath.Rel(prefix, path)
	if err != nil || rel == "." {
		return prefix
	}
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) == 0 {
		return prefix
	}
	return filepath.Join(prefix, parts[0])
}

// CheckLogs checks the total size of /var/log recursively.
func CheckLogs(sysFS system.FileSystem, config model.Config) model.Finding {
	finding := model.Finding{
		ID:           "logs.varlog.size",
		Title:        "Log files size check",
		CheckCommand: "find /var/log -type f -exec du -sh {} +",
	}

	logDir := "/var/log"
	_, err := sysFS.Stat(logDir)
	if err != nil {
		finding.Severity = model.SeveritySkipped
		finding.Title = "Log files size check skipped"
		finding.Summary = "Log files size check could not be run."
		finding.Details = []string{fmt.Sprintf("Failed to access /var/log: %v", err)}
		return finding
	}

	var totalSize int64
	var files []fileInfo
	dirSizes := make(map[string]int64)
	var hasSparseLogs bool

	err = sysFS.WalkDir(logDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		name := d.Name()
		if name == "lastlog" || name == "btmp" || name == "wtmp" {
			hasSparseLogs = true
		}

		size := info.Size()
		totalSize += size
		files = append(files, fileInfo{Path: path, Size: size})

		// Track directory size
		subdir := getVarLogSubdir(path)
		dirSizes[subdir] += size

		return nil
	})

	if err != nil {
		finding.Severity = model.SeveritySkipped
		finding.Title = "Log files size check skipped"
		finding.Summary = "Log files size check could not be run."
		finding.Details = []string{fmt.Sprintf("Walk error: %v", err)}
		return finding
	}

	// Sort files descending
	sort.Slice(files, func(i, j int) bool {
		return files[i].Size > files[j].Size
	})

	// Convert directory sizes to slice and sort descending
	var dirs []dirInfo
	for dPath, dSize := range dirSizes {
		dirs = append(dirs, dirInfo{Path: dPath, Size: dSize})
	}
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].Size > dirs[j].Size
	})

	// Check thresholds
	warningThreshold := config.Logs.VarLogWarningMB * 1024 * 1024
	criticalThreshold := config.Logs.VarLogCriticalMB * 1024 * 1024

	if totalSize >= criticalThreshold {
		finding.Severity = model.SeverityCritical
		finding.Title = "Huge log files detected"
		finding.Summary = fmt.Sprintf("Total log size is %.2f MB (threshold: %d MB).", float64(totalSize)/(1024*1024), config.Logs.VarLogCriticalMB)
		finding.Suggestion = "Inspect the largest logs and fix the source before deleting or truncating files."
	} else if totalSize >= warningThreshold {
		finding.Severity = model.SeverityWarning
		finding.Title = "Large log files detected"
		finding.Summary = fmt.Sprintf("Total log size is %.2f MB (threshold: %d MB).", float64(totalSize)/(1024*1024), config.Logs.VarLogWarningMB)
		finding.Suggestion = "Inspect the largest logs and fix the source before deleting or truncating files."
	} else {
		finding.Severity = model.SeverityOK
		finding.Title = "Log files size looks normal"
		finding.Summary = fmt.Sprintf("Total log size is %.2f MB.", float64(totalSize)/(1024*1024))
	}

	// Format details
	var details []string
	details = append(details, fmt.Sprintf("Total apparent size of /var/log: %.2f MB", float64(totalSize)/(1024*1024)))

	// Top 5 directories
	limitDirs := 5
	if len(dirs) < limitDirs {
		limitDirs = len(dirs)
	}
	if limitDirs > 0 {
		details = append(details, "Largest directories:")
		for i := 0; i < limitDirs; i++ {
			details = append(details, fmt.Sprintf("  - %s: %.2f MB", dirs[i].Path, float64(dirs[i].Size)/(1024*1024)))
		}
	}

	// Top 5 files
	limitFiles := 5
	if len(files) < limitFiles {
		limitFiles = len(files)
	}
	if limitFiles > 0 {
		details = append(details, "Largest files:")
		for i := 0; i < limitFiles; i++ {
			details = append(details, fmt.Sprintf("  - %s: %.2f MB", files[i].Path, float64(files[i].Size)/(1024*1024)))
		}
	}

	if hasSparseLogs {
		details = append(details, "Note: lastlog, btmp, and wtmp may be sparse or misleading. Use du -h for disk usage.")
	}

	finding.Details = details
	return finding
}
