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

func formatSize(bytes int64) string {
	const gb = 1024 * 1024 * 1024
	const mb = 1024 * 1024
	if bytes >= gb {
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(gb))
	}
	return fmt.Sprintf("%.2f MB", float64(bytes)/float64(mb))
}

func isLikelySparse(apparent, actual int64) bool {
	if apparent <= 1024*1024 {
		return false
	}
	return apparent >= actual*10 && apparent > actual
}

// CheckLogs checks the total actual disk size of /var/log recursively.
func CheckLogs(sysFS system.FileSystem, cfg config.Config) model.Finding {
	finding := model.Finding{
		ID:           "logs.varlog.size",
		CheckCommand: "sudo du -h -d 1 /var/log | sort -h",
	}

	logDir := "/var/log"
	_, err := sysFS.Stat(logDir)
	if err != nil {
		finding.Severity = model.SeveritySkipped
		finding.Title = "Log storage check skipped"
		finding.Summary = "Log storage check could not be run."
		finding.Details = []string{fmt.Sprintf("Failed to access /var/log: %v", err)}
		return finding
	}

	var totalSize int64
	var totalActualSize int64
	var files []fileInfo
	dirSizes := make(map[string]int64)
	dirActualSizes := make(map[string]int64)
	var sparseFiles []string
	visited := make(map[string]bool)

	err = sysFS.WalkDir(logDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}

		if d.Type()&fs.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(path)
			if err == nil {
				if visited[target] {
					return fs.SkipDir
				}
				visited[target] = true
			}
			return nil
		}

		if !d.Type().IsRegular() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		size := info.Size()
		actualSize := sysFS.ActualSize(info)

		totalSize += size
		totalActualSize += actualSize

		files = append(files, fileInfo{Path: path, Size: size, ActualSize: actualSize})

		subdir := getVarLogSubdir(path)
		dirSizes[subdir] += size
		dirActualSizes[subdir] += actualSize

		if isLikelySparse(size, actualSize) {
			sparseFiles = append(sparseFiles, path)
		}

		return nil
	})

	if err != nil {
		finding.Severity = model.SeveritySkipped
		finding.Title = "Log storage check skipped"
		finding.Summary = "Log storage check could not be run."
		finding.Details = []string{fmt.Sprintf("Walk error: %v", err)}
		return finding
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].ActualSize > files[j].ActualSize
	})

	var dirs []dirInfo
	for dPath, dSize := range dirActualSizes {
		dirs = append(dirs, dirInfo{Path: dPath, Size: dirSizes[dPath], ActualSize: dSize})
	}
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].ActualSize > dirs[j].ActualSize
	})

	warningThreshold := cfg.Logs.WarningMB * 1024 * 1024
	criticalThreshold := cfg.Logs.CriticalMB * 1024 * 1024

	if totalActualSize >= criticalThreshold {
		finding.Severity = model.SeverityCritical
		finding.Title = "Large log storage detected"
		finding.Summary = fmt.Sprintf("/var/log uses %s on disk.", formatSize(totalActualSize))
		finding.Suggestion = "Inspect the largest logs and fix the source before deleting or truncating files."
	} else if totalActualSize >= warningThreshold {
		finding.Severity = model.SeverityWarning
		finding.Title = "Large log storage detected"
		finding.Summary = fmt.Sprintf("/var/log uses %s on disk.", formatSize(totalActualSize))
		finding.Suggestion = "Inspect the largest logs and fix the source before deleting or truncating files."
	} else {
		finding.Severity = model.SeverityOK
		finding.Title = "Log storage looks normal"
		finding.Summary = fmt.Sprintf("/var/log uses %s on disk.", formatSize(totalActualSize))
	}

	var details []string
	details = append(details, fmt.Sprintf("Actual disk usage: %s", formatSize(totalActualSize)))
	details = append(details, fmt.Sprintf("Apparent size: %s", formatSize(totalSize)))

	limitDirs := 5
	if len(dirs) < limitDirs {
		limitDirs = len(dirs)
	}
	if limitDirs > 0 {
		details = append(details, "Largest directories:")
		for i := 0; i < limitDirs; i++ {
			details = append(details, fmt.Sprintf("  - %s: %s", dirs[i].Path, formatSize(dirs[i].ActualSize)))
		}
	}

	limitFiles := 5
	if len(files) < limitFiles {
		limitFiles = len(files)
	}
	if limitFiles > 0 {
		details = append(details, "Largest files:")
		for i := 0; i < limitFiles; i++ {
			details = append(details, fmt.Sprintf("  - %s: %s", files[i].Path, formatSize(files[i].ActualSize)))
		}
	}

	if len(sparseFiles) > 0 {
		details = append(details, "Sparse files detected:")
		limit := 5
		if len(sparseFiles) < limit {
			limit = len(sparseFiles)
		}
		for i := 0; i < limit; i++ {
			details = append(details, fmt.Sprintf("  - %s appears sparse; apparent size may be misleading.", sparseFiles[i]))
		}
		if len(sparseFiles) > limit {
			details = append(details, fmt.Sprintf("  - ... and %d more", len(sparseFiles)-limit))
		}
	}

	finding.Details = details
	return finding
}
