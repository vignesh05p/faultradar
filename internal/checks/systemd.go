package checks

import (
	"fmt"
	"path"
	"strings"

	"faultradar/internal/config"
	"faultradar/internal/model"
	"faultradar/internal/system"
)

var defaultImportantUnits = []string{
	"mysql.service",
	"postgresql.service",
	"docker.service",
	"containerd.service",
	"ssh.service",
	"sshd.service",
	"NetworkManager.service",
	"gdm.service",
	"sddm.service",
	"lightdm.service",
	"display-manager.service",
}

func isImportantUnit(unitName string, importantUnits []string) bool {
	for _, imp := range importantUnits {
		if imp == unitName {
			return true
		}
		matched, err := path.Match(imp, unitName)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func isIgnoredUnit(unitName string, cfg config.Config) bool {
	for _, iu := range cfg.Systemd.IgnoreUnits {
		if iu == unitName {
			return true
		}
	}
	for _, ip := range cfg.Systemd.IgnoreUnitPatterns {
		matched, err := path.Match(ip, unitName)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func parseFailedUnits(output string, cfg config.Config) (services, snapMounts, mounts, timers, sockets, others []string) {
	importantUnits := cfg.Systemd.ImportantUnits
	if len(importantUnits) == 0 {
		importantUnits = defaultImportantUnits
	}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		if fields[0] == "UNIT" || strings.HasPrefix(fields[0], "●") {
			continue
		}
		if fields[2] != "failed" && fields[3] != "failed" {
			continue
		}

		unitName := fields[0]
		if isIgnoredUnit(unitName, cfg) {
			continue
		}

		switch {
		case strings.HasPrefix(unitName, "snap-") && strings.HasSuffix(unitName, ".mount"):
			snapMounts = append(snapMounts, unitName)
		case strings.HasSuffix(unitName, ".service"):
			services = append(services, unitName)
		case strings.HasSuffix(unitName, ".mount"):
			mounts = append(mounts, unitName)
		case strings.HasSuffix(unitName, ".timer"):
			timers = append(timers, unitName)
		case strings.HasSuffix(unitName, ".socket"):
			sockets = append(sockets, unitName)
		default:
			others = append(others, unitName)
		}
	}
	return
}

// CheckSystemd checks for failed systemd units.
func CheckSystemd(runner system.CommandRunner, cfg config.Config) []model.Finding {
	checkCmd := "systemctl --failed --no-pager --plain"

	outputBytes, err := runner.Run("systemctl", "--failed", "--no-pager", "--plain")
	output := string(outputBytes)

	if err != nil && !strings.Contains(output, "loaded units listed") {
		return []model.Finding{{
			ID:           "systemd.unavailable",
			Severity:     model.SeveritySkipped,
			Title:        "Systemd diagnostics unavailable",
			Summary:      "systemctl is unavailable or systemd is not running.",
			CheckCommand: checkCmd,
			Details:      []string{fmt.Sprintf("Error: %v", err)},
		}}
	}

	services, snapMounts, mounts, timers, sockets, others := parseFailedUnits(output, cfg)
	importantUnits := cfg.Systemd.ImportantUnits
	if len(importantUnits) == 0 {
		importantUnits = defaultImportantUnits
	}

	var findings []model.Finding

	var importantFailed []string
	var normalServices []string
	for _, s := range services {
		if isImportantUnit(s, importantUnits) {
			importantFailed = append(importantFailed, s)
		} else {
			normalServices = append(normalServices, s)
		}
	}

	for _, unit := range importantFailed {
		findings = append(findings, model.Finding{
			ID:           "systemd.failed.important",
			Severity:     model.SeverityCritical,
			Title:        "Important systemd service failed",
			Summary:      fmt.Sprintf("%s is in failed state.", unit),
			Suggestion:   "Inspect the failed service and its logs.",
			CheckCommand: fmt.Sprintf("systemctl status %s", unit),
		})
	}

	if len(normalServices) > 0 {
		findings = append(findings, model.Finding{
			ID:           "systemd.failed.services",
			Severity:     model.SeverityWarning,
			Title:        "Failed systemd services detected",
			Summary:      fmt.Sprintf("%d failed service unit(s) detected.", len(normalServices)),
			Suggestion:   "Inspect the failed services and their logs.",
			CheckCommand: checkCmd,
			Details:      formatGroup("Examples", normalServices, 3),
		})
	}

	if len(snapMounts) > 0 {
		findings = append(findings, model.Finding{
			ID:           "systemd.failed.snap_mounts",
			Severity:     model.SeverityWarning,
			Title:        "Snap mount failures detected",
			Summary:      fmt.Sprintf("%d failed Snap mount unit(s) detected.", len(snapMounts)),
			Suggestion:   "These are often stale Snap mount units after package updates. Inspect before ignoring.",
			CheckCommand: "systemctl --failed --type=mount --no-pager",
			Details:      formatGroup("Examples", snapMounts, 2),
		})
	}

	otherUnits := append(append(append(mounts, timers...), sockets...), others...)
	if len(otherUnits) > 0 {
		findings = append(findings, model.Finding{
			ID:           "systemd.failed.other",
			Severity:     model.SeverityWarning,
			Title:        "Other failed systemd units detected",
			Summary:      fmt.Sprintf("%d failed non-service unit(s) detected.", len(otherUnits)),
			Suggestion:   "Inspect the failed units and their logs.",
			CheckCommand: checkCmd,
			Details:      formatGroup("Examples", otherUnits, 3),
		})
	}

	totalFailures := len(services) + len(snapMounts) + len(mounts) + len(timers) + len(sockets) + len(others)
	if totalFailures == 0 {
		findings = append(findings, model.Finding{
			ID:           "systemd.failed.none",
			Severity:     model.SeverityOK,
			Title:        "No failed systemd units found",
			Summary:      "All systemd units are running normally.",
			CheckCommand: checkCmd,
		})
	}

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
