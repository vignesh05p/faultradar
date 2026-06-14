package checks

import (
	"fmt"
	"regexp"
	"strings"

	"faultradar/internal/config"
	"faultradar/internal/model"
	"faultradar/internal/system"
)

var criticalPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)I/O error`),
	regexp.MustCompile(`(?i)Buffer I/O error`),
	regexp.MustCompile(`(?i)EXT4-fs error`),
	regexp.MustCompile(`(?i)XFS.*error`),
	regexp.MustCompile(`(?i)BTRFS.*error`),
	regexp.MustCompile(`(?i)nvme.*timeout`),
	regexp.MustCompile(`(?i)ata.*error`),
	regexp.MustCompile(`(?i)filesystem.*read-only`),
	regexp.MustCompile(`(?i)Out of memory`),
	regexp.MustCompile(`(?i)oom-killer`),
	regexp.MustCompile(`(?i)kernel panic`),
	regexp.MustCompile(`(?i)BUG:`),
	regexp.MustCompile(`(?i)soft lockup`),
	regexp.MustCompile(`(?i)hard lockup`),
	regexp.MustCompile(`(?i)watchdog.*lockup`),
}

var warningPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)ACPI Error`),
	regexp.MustCompile(`(?i)ACPI BIOS Error`),
	regexp.MustCompile(`(?i)snap.*Can't lookup blockdev`),
	regexp.MustCompile(`(?i)blockdev.*snap`),
	regexp.MustCompile(`(?i)Can't lookup blockdev`),
}

type compiledPatterns struct {
	valid   []*regexp.Regexp
	invalid []string
}

func compilePatterns(name string, patterns []string) compiledPatterns {
	var result compiledPatterns
	for _, p := range patterns {
		re, err := regexp.Compile("(?i)" + p)
		if err != nil {
			result.invalid = append(result.invalid, fmt.Sprintf("%s ignore pattern %q is invalid: %v", name, p, err))
			continue
		}
		result.valid = append(result.valid, re)
	}
	return result
}

func isRestrictedKernelOutput(output string, err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(output + " " + err.Error())
	return strings.Contains(lower, "permission") ||
		strings.Contains(lower, "access denied") ||
		strings.Contains(lower, "not permitted") ||
		strings.Contains(lower, "insufficient privileges")
}

func truncateLine(line string, maxLen int) string {
	if len(line) <= maxLen {
		return line
	}
	return line[:maxLen] + "..."
}

// CheckKernel checks for kernel errors in the current boot.
func CheckKernel(runner system.CommandRunner, cfg config.Config) model.Finding {
	checkCmd := "journalctl -k -p 3 -b --no-pager"

	outputBytes, err := runner.Run("journalctl", "-k", "-p", "3", "-b", "--no-pager")
	output := string(outputBytes)

	if err != nil {
		if isRestrictedKernelOutput(output, err) {
			return model.Finding{
				ID:           "kernel.restricted",
				Severity:     model.SeveritySkipped,
				Title:        "Kernel log check restricted",
				Summary:      "Kernel journal access is restricted for the current user.",
				CheckCommand: checkCmd,
				Details:      []string{fmt.Sprintf("Error: %v", err)},
			}
		}
		return model.Finding{
			ID:           "kernel.unavailable",
			Severity:     model.SeveritySkipped,
			Title:        "Kernel log check unavailable",
			Summary:      "journalctl is unavailable or could not read kernel logs.",
			CheckCommand: checkCmd,
			Details:      []string{fmt.Sprintf("Error: %v", err)},
		}
	}

	ignoreCompiled := compilePatterns("kernel", cfg.Kernel.IgnorePatterns)
	downgradeCompiled := compilePatterns("kernel downgrade", cfg.Kernel.DowngradePatterns)

	var criticalLines []string
	var warningLines []string

	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		ignored := false
		for _, re := range ignoreCompiled.valid {
			if re.MatchString(trimmed) {
				ignored = true
				break
			}
		}
		if ignored {
			continue
		}

		isCritical := false
		for _, re := range criticalPatterns {
			if re.MatchString(trimmed) {
				isCritical = true
				break
			}
		}

		if isCritical {
			downgraded := false
			for _, re := range downgradeCompiled.valid {
				if re.MatchString(trimmed) {
					downgraded = true
					break
				}
			}
			if downgraded {
				warningLines = append(warningLines, trimmed)
			} else {
				criticalLines = append(criticalLines, trimmed)
			}
			continue
		}

		isWarning := false
		for _, re := range warningPatterns {
			if re.MatchString(trimmed) {
				isWarning = true
				break
			}
		}
		if isWarning {
			warningLines = append(warningLines, trimmed)
		}
	}

	if len(criticalLines) == 0 && len(warningLines) == 0 {
		finding := model.Finding{
			ID:           "kernel.errors.none",
			Severity:     model.SeverityOK,
			Title:        "No kernel errors found in current boot",
			Summary:      "No relevant kernel priority-3 messages were found in this boot.",
			CheckCommand: checkCmd,
		}
		if len(ignoreCompiled.invalid) > 0 || len(downgradeCompiled.invalid) > 0 {
			finding.Details = append(ignoreCompiled.invalid, downgradeCompiled.invalid...)
		}
		return finding
	}

	var finding model.Finding
	finding.CheckCommand = checkCmd

	if len(criticalLines) > 0 {
		finding.ID = "kernel.errors.critical"
		finding.Severity = model.SeverityCritical
		finding.Title = "Critical kernel errors found in current boot"
		finding.Summary = fmt.Sprintf("%d critical kernel message(s) found.", len(criticalLines))
		finding.Suggestion = "Inspect kernel logs for disk, filesystem, or hardware problems."
		finding.Details = appendExampleLines("Critical examples", criticalLines, 3)
	} else {
		finding.ID = "kernel.errors.warning"
		finding.Severity = model.SeverityWarning
		finding.Title = "Kernel warnings found in current boot"
		finding.Summary = fmt.Sprintf("%d warning-like kernel message(s) found.", len(warningLines))
		finding.Suggestion = "Inspect kernel logs if you are seeing hardware, driver, boot, or Snap issues."
		finding.Details = appendExampleLines("Warning examples", warningLines, 3)
	}

	if len(ignoreCompiled.invalid) > 0 || len(downgradeCompiled.invalid) > 0 {
		finding.Details = append(finding.Details, ignoreCompiled.invalid...)
		finding.Details = append(finding.Details, downgradeCompiled.invalid...)
	}

	return finding
}

func appendExampleLines(header string, lines []string, limit int) []string {
	if len(lines) == 0 {
		return nil
	}
	var details []string
	details = append(details, header+":")
	if len(lines) <= limit {
		for _, line := range lines {
			details = append(details, "  - "+truncateLine(line, 120))
		}
	} else {
		for i := 0; i < limit; i++ {
			details = append(details, "  - "+truncateLine(lines[i], 120))
		}
		details = append(details, fmt.Sprintf("  - ... and %d more", len(lines)-limit))
	}
	return details
}

// ValidateKernelPatterns returns findings for invalid config regex patterns.
func ValidateKernelPatterns(cfg config.Config) []model.Finding {
	var findings []model.Finding
	for _, p := range cfg.Kernel.IgnorePatterns {
		if _, err := regexp.Compile("(?i)" + p); err != nil {
			findings = append(findings, model.Finding{
				ID:       "config.kernel.ignore_pattern",
				Severity: model.SeverityWarning,
				Title:    "Invalid kernel ignore pattern in config",
				Summary:  fmt.Sprintf("Pattern %q could not be compiled.", p),
				Details:  []string{err.Error()},
			})
		}
	}
	for _, p := range cfg.Kernel.DowngradePatterns {
		if _, err := regexp.Compile("(?i)" + p); err != nil {
			findings = append(findings, model.Finding{
				ID:       "config.kernel.downgrade_pattern",
				Severity: model.SeverityWarning,
				Title:    "Invalid kernel downgrade pattern in config",
				Summary:  fmt.Sprintf("Pattern %q could not be compiled.", p),
				Details:  []string{err.Error()},
			})
		}
	}
	return findings
}
