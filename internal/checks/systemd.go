package checks

import (
	"fmt"
	"path"
	"strings"

	"faultradar/internal/config"
	"faultradar/internal/model"
	"faultradar/internal/system"
)

// CheckSystemd checks for failed systemd units, returning separate findings for services/other units and snap mounts.
func CheckSystemd(runner system.CommandRunner, cfg config.Config) []model.Finding {
	failedUnitsFinding := model.Finding{
		ID:           "systemd.failed_units",
		Title:        "Failed systemd services check",
		CheckCommand: "systemctl --failed --no-pager --plain",
	}

	failedSnapFinding := model.Finding{
		ID:           "systemd.failed_snap_mounts",
		Title:        "Failed snap mount units check",
		CheckCommand: "systemctl --failed --no-pager --plain",
	}

	outputBytes, err := runner.Run("systemctl", "--failed", "--no-pager", "--plain")
	output := string(outputBytes)

	if err != nil && !strings.Contains(output, "loaded units listed") {
		failedUnitsFinding.Severity = model.SeveritySkipped
		failedUnitsFinding.Title = "Systemd diagnostics unavailable"
		failedUnitsFinding.Summary = "systemctl command was not found or failed to execute."
		failedUnitsFinding.Details = []string{fmt.Sprintf("Error: %v", err)}

		failedSnapFinding.Severity = model.SeveritySkipped
		failedSnapFinding.Title = "Snap mount diagnostics unavailable"
		failedSnapFinding.Summary = "systemctl command was not found or failed to execute."
		failedSnapFinding.Details = []string{fmt.Sprintf("Error: %v", err)}

		return []model.Finding{failedUnitsFinding, failedSnapFinding}
	}

	lines := strings.Split(output, "\n")
	var services []string
	var snapMounts []string
	var mounts []string
	var timers []string
	var sockets []string
	var others []string

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

			// Check ignores
			ignored := false
			for _, iu := range cfg.Systemd.IgnoreUnits {
				if iu == unitName {
					ignored = true
					break
				}
			}
			if !ignored {
				for _, ip := range cfg.Systemd.IgnoreUnitPatterns {
					matched, err := path.Match(ip, unitName)
					if err == nil && matched {
						ignored = true
						break
					}
				}
			}

			if ignored {
				continue
			}

			// Categorize
			if strings.HasPrefix(unitName, "snap-") && strings.HasSuffix(unitName, ".mount") {
				snapMounts = append(snapMounts, unitName)
			} else if strings.HasSuffix(unitName, ".service") {
				services = append(services, unitName)
			} else if strings.HasSuffix(unitName, ".mount") {
				mounts = append(mounts, unitName)
			} else if strings.HasSuffix(unitName, ".timer") {
				timers = append(timers, unitName)
			} else if strings.HasSuffix(unitName, ".socket") {
				sockets = append(sockets, unitName)
			} else {
				others = append(others, unitName)
			}
		}
	}

	var findings []model.Finding

	// 1. services & standard units finding
	totalFailedUnits := len(services) + len(mounts) + len(timers) + len(sockets) + len(others)
	if totalFailedUnits == 0 {
		failedUnitsFinding.Severity = model.SeverityOK
		failedUnitsFinding.Title = "No failed systemd services found"
		failedUnitsFinding.Summary = "All services and system units are running normally."
	} else {
		hasImportantFailure := false
		for _, s := range services {
			for _, imp := range cfg.Systemd.ImportantUnits {
				if imp == s {
					hasImportantFailure = true
					break
				}
				matched, err := path.Match(imp, s)
				if err == nil && matched {
					hasImportantFailure = true
					break
				}
			}
			if hasImportantFailure {
				break
			}
		}

		if hasImportantFailure {
			failedUnitsFinding.Severity = model.SeverityCritical
		} else {
			failedUnitsFinding.Severity = model.SeverityWarning
		}

		failedUnitsFinding.Title = "Failed systemd services found"
		failedUnitsFinding.Summary = fmt.Sprintf("%d failed systemd unit(s) detected.", totalFailedUnits)
		failedUnitsFinding.Suggestion = "Inspect failed services and their logs."

		var details []string
		details = append(details, formatGroup("Failed services", services, 10)...)
		details = append(details, formatGroup("Failed mount units", mounts, 10)...)
		details = append(details, formatGroup("Failed timer units", timers, 10)...)
		details = append(details, formatGroup("Failed socket units", sockets, 10)...)
		details = append(details, formatGroup("Failed other units", others, 10)...)
		failedUnitsFinding.Details = details
	}
	findings = append(findings, failedUnitsFinding)

	// 2. snap mount units finding
	if len(snapMounts) == 0 {
		failedSnapFinding.Severity = model.SeverityOK
		failedSnapFinding.Title = "No failed snap mount units found"
		failedSnapFinding.Summary = "All snap mount points are mounted correctly."
	} else {
		failedSnapFinding.Severity = model.SeverityWarning
		failedSnapFinding.Title = "Failed snap mount units found"
		failedSnapFinding.Summary = fmt.Sprintf("%d failed snap mount unit(s) detected.", len(snapMounts))
		failedSnapFinding.Suggestion = "These may be temporary or noisy snap environment issues. Check snapd status if persistent."

		// Limit snap mounts listed in details to 3 examples as requested
		failedSnapFinding.Details = formatGroup("Failed snap mount units", snapMounts, 3)
	}
	findings = append(findings, failedSnapFinding)

	return findings
}

func formatGroup(groupName string, units []string, limit int) []string {
	if len(units) == 0 {
		return nil
	}
	var res []string
	res = append(res, groupName+":")
	if len(units) <= limit {
		for _, u := range units {
			res = append(res, "  - "+u)
		}
	} else {
		for i := 0; i < limit; i++ {
			res = append(res, "  - "+units[i])
		}
		res = append(res, fmt.Sprintf("  - ... and %d more", len(units)-limit))
	}
	return res
}
