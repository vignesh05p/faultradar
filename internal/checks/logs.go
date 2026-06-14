package checks

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"faultradar/internal/config"
	"faultradar/internal/model"
	"faultradar/internal/system"
)

type fileInfo struct {
	Path       string
	Size       int64
	ActualSize int64
}

type dirInfo struct {
	Path       string
	Size       int64
	ActualSize int64
}

func getVarLogSubdir(path string) string {
	const prefix = "/var/log"
	if path == prefix {
		return prefix
	}
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

// CheckLogs checks the total actual disk size of /var/log recursively.
func CheckLogs(sysFS system.FileSystem, cfg config.Config) model.Finding {
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
	var totalActualSize int64
	var files []fileInfo
	dirSizes := make(map[string]int64)
	dirActualSizes := make(map[string]int64)
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
		actualSize := sysFS.ActualSize(info)

		totalSize += size
		totalActualSize += actualSize

		files = append(files, fileInfo{Path: path, Size: size, ActualSize: actualSize})

		subdir := getVarLogSubdir(path)
		dirSizes[subdir] += size
		dirActualSizes[subdir] += actualSize

		return nil
	})

	if err != nil {
		finding.Severity = model.SeveritySkipped
		finding.Title = "Log files size check skipped"
		finding.Summary = "Log files size check could not be run."
		finding.Details = []string{fmt.Sprintf("Walk error: %v", err)}
		return finding
	}

	// Sort files by actual size descending
	sort.Slice(files, func(i, j int) bool {
		return files[i].ActualSize > files[j].ActualSize
	})

	// Convert directory sizes to slice and sort descending by actual size
	var dirs []dirInfo
	for dPath, dSize := range dirActualSizes {
		dirs = append(dirs, dirInfo{Path: dPath, Size: dirSizes[dPath], ActualSize: dSize})
	}
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].ActualSize > dirs[j].ActualSize
	})

	// Check thresholds using actual disk size
	warningThreshold := cfg.Logs.VarLogWarningMB * 1024 * 1024
	criticalThreshold := cfg.Logs.VarLogCriticalMB * 1024 * 1024

	if totalActualSize >= criticalThreshold {
		finding.Severity = model.SeverityCritical
		finding.Title = "Huge log files detected"
		finding.Summary = fmt.Sprintf("Total actual log size is %.2f MB (threshold: %d MB).", float64(totalActualSize)/(1024*1024), cfg.Logs.VarLogCriticalMB)
		finding.Suggestion = "Inspect the largest logs and fix the source before deleting or truncating files."
	} else if totalActualSize >= warningThreshold {
		finding.Severity = model.SeverityWarning
		finding.Title = "Large log files detected"
		finding.Summary = fmt.Sprintf("Total actual log size is %.2f MB (threshold: %d MB).", float64(totalActualSize)/(1024*1024), cfg.Logs.VarLogWarningMB)
		finding.Suggestion = "Inspect the largest logs and fix the source before deleting or truncating files."
	} else {
		finding.Severity = model.SeverityOK
		finding.Title = "Log files size looks normal"
		finding.Summary = fmt.Sprintf("Total actual log size is %.2f MB.", float64(totalActualSize)/(1024*1024))
	}

	// Format details
	var details []string
	details = append(details, fmt.Sprintf("Total actual size of /var/log: %.2f MB (apparent size: %.2f MB)", float64(totalActualSize)/(1024*1024), float64(totalSize)/(1024*1024)))

	// Top 5 directories by actual size
	limitDirs := 5
	if len(dirs) < limitDirs {
		limitDirs = len(dirs)
	}
	if limitDirs > 0 {
		details = append(details, "Largest directories:")
		for i := 0; i < limitDirs; i++ {
			if dirs[i].ActualSize != dirs[i].Size {
				details = append(details, fmt.Sprintf("  - %s: %.2f MB (apparent: %.2f MB)", dirs[i].Path, float64(dirs[i].ActualSize)/(1024*1024), float64(dirs[i].Size)/(1024*1024)))
			} else {
				details = append(details, fmt.Sprintf("  - %s: %.2f MB", dirs[i].Path, float64(dirs[i].ActualSize)/(1024*1024)))
			}
		}
	}

	// Top 5 files by actual size
	limitFiles := 5
	if len(files) < limitFiles {
		limitFiles = len(files)
	}
	if limitFiles > 0 {
		details = append(details, "Largest files:")
		for i := 0; i < limitFiles; i++ {
			if files[i].ActualSize != files[i].Size {
				details = append(details, fmt.Sprintf("  - %s: %.2f MB (apparent: %.2f MB)", files[i].Path, float64(files[i].ActualSize)/(1024*1024), float64(files[i].Size)/(1024*1024)))
			} else {
				details = append(details, fmt.Sprintf("  - %s: %.2f MB", files[i].Path, float64(files[i].ActualSize)/(1024*1024)))
			}
		}
	}

	if hasSparseLogs {
		details = append(details, "Note: lastlog, btmp, and wtmp may be sparse or misleading. Use du -h for disk usage.")
	}

	finding.Details = details
	return finding
}
