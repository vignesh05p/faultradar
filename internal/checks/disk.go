package checks

import (
	"fmt"

	"faultradar/internal/config"
	"faultradar/internal/model"
	"faultradar/internal/system"
)

// CheckDisk checks the root filesystem disk usage.
func CheckDisk(fs system.FileSystem, cfg config.Config) model.Finding {
	finding := model.Finding{
		ID:           "disk.root.usage",
		CheckCommand: "df -h /",
	}

	stat, err := fs.Statfs("/")
	if err != nil {
		finding.Severity = model.SeveritySkipped
		finding.Title = "Root disk usage check skipped"
		finding.Summary = "Root disk usage check could not be completed."
		finding.Details = []string{fmt.Sprintf("Failed to query root filesystem status: %v", err)}
		return finding
	}

	used := stat.Blocks - stat.Bfree
	totalUsable := used + stat.Bavail

	var percent int
	if totalUsable > 0 {
		percent = int((used*100 + totalUsable - 1) / totalUsable)
	}

	warningThreshold := cfg.Disk.WarningPercent
	criticalThreshold := cfg.Disk.CriticalPercent

	if percent >= criticalThreshold {
		finding.Severity = model.SeverityCritical
		finding.Title = "Root disk is almost full"
		finding.Summary = fmt.Sprintf("Root filesystem is %d%% used.", percent)
		finding.Suggestion = "Free disk space before updates or apps start failing."
	} else if percent >= warningThreshold {
		finding.Severity = model.SeverityWarning
		finding.Title = "Root disk usage is high"
		finding.Summary = fmt.Sprintf("Root filesystem is %d%% used.", percent)
		finding.Suggestion = "Free disk space before updates or apps start failing."
	} else {
		finding.Severity = model.SeverityOK
		finding.Title = "Root disk usage looks normal"
		finding.Summary = fmt.Sprintf("Root disk usage is %d%%.", percent)
	}

	finding.Details = []string{
		fmt.Sprintf("Mount: /"),
		fmt.Sprintf("Used: %d%%", percent),
	}
	return finding
}
