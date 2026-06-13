package checks

import (
	"fmt"
	"io/fs"
	"sort"

	"faultradar/internal/model"
	"faultradar/internal/system"
)

type fileInfo struct {
	Path string
	Size int64
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
		finding.Summary = "/var/log directory is inaccessible."
		finding.Details = []string{fmt.Sprintf("Failed to access %s: %v", logDir, err)}
		return finding
	}

	var totalSize int64
	var files []fileInfo

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
		size := info.Size()
		totalSize += size
		files = append(files, fileInfo{Path: path, Size: size})
		return nil
	})

	if err != nil {
		finding.Severity = model.SeveritySkipped
		finding.Title = "Log files size check skipped"
		finding.Summary = "Failed to walk /var/log directory."
		finding.Details = []string{fmt.Sprintf("Walk error: %v", err)}
		return finding
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Size > files[j].Size
	})

	var details []string
	topCount := 5
	if len(files) < topCount {
		topCount = len(files)
	}

	details = append(details, fmt.Sprintf("Total size of %s: %.2f MB", logDir, float64(totalSize)/(1024*1024)))
	if topCount > 0 {
		details = append(details, "Largest log files:")
		for i := 0; i < topCount; i++ {
			details = append(details, fmt.Sprintf("  - %s: %.2f MB", files[i].Path, float64(files[i].Size)/(1024*1024)))
		}
	}

	warningThreshold := config.Logs.VarLogWarningMB * 1024 * 1024
	criticalThreshold := config.Logs.VarLogCriticalMB * 1024 * 1024

	if totalSize >= criticalThreshold {
		finding.Severity = model.SeverityCritical
		finding.Title = "Huge log files detected"
		finding.Summary = fmt.Sprintf("Total log size is %.1f GB (threshold: %d GB).", float64(totalSize)/1e9, config.Logs.VarLogCriticalMB/1024)
		finding.Suggestion = "A repeated system or kernel error may be filling these files. Inspect before deleting anything."
	} else if totalSize >= warningThreshold {
		finding.Severity = model.SeverityWarning
		finding.Title = "Large log files detected"
		finding.Summary = fmt.Sprintf("Total log size is %.1f GB (threshold: %d GB).", float64(totalSize)/1e9, config.Logs.VarLogWarningMB/1024)
		finding.Suggestion = "Inspect largest log files and resolve system errors or rotate logs."
	} else {
		finding.Severity = model.SeverityOK
		finding.Title = "Log files size looks normal"
		finding.Summary = fmt.Sprintf("Total log size is %.2f MB.", float64(totalSize)/(1024*1024))
	}

	finding.Details = details
	return finding
}
