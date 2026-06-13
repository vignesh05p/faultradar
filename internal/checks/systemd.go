package checks

import (
	"fmt"
	"strings"

	"faultradar/internal/model"
	"faultradar/internal/system"
)

// CheckSystemd checks for failed systemd units, respecting config ignores.
func CheckSystemd(runner system.CommandRunner, config model.Config) model.Finding {
	finding := model.Finding{
		ID:           "systemd.failed_units",
		Title:        "Failed systemd services check",
		CheckCommand: "systemctl --failed --no-pager --plain",
	}

	outputBytes, err := runner.Run("systemctl", "--failed", "--no-pager", "--plain")
	output := string(outputBytes)

	// If there's an error and output doesn't contain systemctl markers, we assume run failed.
	if err != nil && !strings.Contains(output, "loaded units listed") {
		errStr := err.Error()
		if strings.Contains(errStr, "not found") || strings.Contains(errStr, "no such file") || strings.Contains(errStr, "executable file not found") {
			finding.Severity = model.SeveritySkipped
			finding.Title = "Systemd diagnostics unavailable"
			finding.Summary = "systemctl command was not found on this system."
			finding.Details = []string{fmt.Sprintf("Error: %v", err)}
			return finding
		}
		finding.Severity = model.SeverityWarning
		finding.Title = "Failed systemd services check error"
		finding.Summary = "Failed to run systemctl command."
		finding.Details = []string{fmt.Sprintf("Run error: %v", err)}
		return finding
	}

	lines := strings.Split(output, "\n")
	var failedUnits []string

	ignoreMap := make(map[string]bool)
	for _, unit := range config.Systemd.IgnoreUnits {
		ignoreMap[unit] = true
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		// Columns: UNIT LOAD ACTIVE SUB DESCRIPTION
		if fields[2] == "failed" || fields[3] == "failed" {
			unitName := fields[0]
			if ignoreMap[unitName] {
				continue
			}
			failedUnits = append(failedUnits, unitName)
		}
	}

	if len(failedUnits) > 0 {
		finding.Severity = model.SeverityWarning
		finding.Title = "Failed systemd services found"
		finding.Summary = fmt.Sprintf("%d failed systemd unit(s) detected.", len(failedUnits))
		finding.Suggestion = "Inspect failed services and their logs."
		finding.Details = append([]string{"Failed units:"}, failedUnits...)
	} else {
		finding.Severity = model.SeverityOK
		finding.Title = "No failed systemd services found"
		finding.Summary = "All systemd units are running normally."
	}

	return finding
}
