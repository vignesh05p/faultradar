package checks

import (
	"fmt"
	"path"
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

	if err != nil && !strings.Contains(output, "loaded units listed") {
		finding.Severity = model.SeveritySkipped
		finding.Title = "Systemd diagnostics unavailable"
		finding.Summary = "systemctl command was not found or failed to execute."
		finding.Details = []string{fmt.Sprintf("Error: %v", err)}
		return finding
	}

	lines := strings.Split(output, "\n")
	var services []string
	var snapMounts []string
	var mounts []string
	var timers []string
	var sockets []string
	var others []string

	var totalFailed int

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
			for _, iu := range config.Systemd.IgnoreUnits {
				if iu == unitName {
					ignored = true
					break
				}
			}
			if !ignored {
				for _, ip := range config.Systemd.IgnoreUnitPatterns {
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

			totalFailed++

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

	if totalFailed == 0 {
		finding.Severity = model.SeverityOK
		finding.Title = "No failed systemd services found"
		finding.Summary = "All systemd units are running normally."
		return finding
	}

	// Determine if any important services failed
	hasImportantFailure := false
	for _, s := range services {
		for _, imp := range config.Systemd.ImportantUnits {
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
		finding.Severity = model.SeverityCritical
	} else {
		finding.Severity = model.SeverityWarning
	}

	finding.Title = "Failed systemd services found"
	finding.Summary = fmt.Sprintf("%d failed systemd unit(s) detected.", totalFailed)
	finding.Suggestion = "Inspect failed services and their logs."

	// Format details
	var details []string
	details = append(details, formatGroup("Failed services", services)...)
	details = append(details, formatGroup("Failed snap mount units", snapMounts)...)
	details = append(details, formatGroup("Failed mount units", mounts)...)
	details = append(details, formatGroup("Failed timer units", timers)...)
	details = append(details, formatGroup("Failed socket units", sockets)...)
	details = append(details, formatGroup("Failed other units", others)...)

	finding.Details = details
	return finding
}

func formatGroup(groupName string, units []string) []string {
	if len(units) == 0 {
		return nil
	}
	var res []string
	res = append(res, groupName+":")
	limit := 10
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
