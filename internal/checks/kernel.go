package checks

import (
	"fmt"
	"regexp"
	"strings"

	"faultradar/internal/model"
	"faultradar/internal/system"
)

var criticalPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)I/O error`),
	regexp.MustCompile(`(?i)Buffer I/O error`),
	regexp.MustCompile(`(?i)blk_update_request`),
	regexp.MustCompile(`(?i)EXT4-fs error`),
	regexp.MustCompile(`(?i)XFS.*corruption`),
	regexp.MustCompile(`(?i)BTRFS.*error`),
	regexp.MustCompile(`(?i)nvme.*timeout`),
	regexp.MustCompile(`(?i)nvme.*I/O`),
	regexp.MustCompile(`(?i)ata.*failed command`),
	regexp.MustCompile(`(?i)filesystem.*read-only`),
	regexp.MustCompile(`(?i)Remounting filesystem read-only`),
	regexp.MustCompile(`(?i)end_request: I/O error`),
	regexp.MustCompile(`(?i)Out of memory`),
	regexp.MustCompile(`(?i)oom-killer`),
	regexp.MustCompile(`(?i)watchdog: BUG: soft lockup`),
	regexp.MustCompile(`(?i)watchdog: Watchdog detected hard LOCKUP`),
	regexp.MustCompile(`(?i)kernel panic`),
	regexp.MustCompile(`(?i)BUG: unable to handle kernel`),
	regexp.MustCompile(`(?i)Machine check events logged`),
	regexp.MustCompile(`(?i)mce: Hardware Error`),
}

var warningPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)ACPI Error`),
	regexp.MustCompile(`(?i)ACPI BIOS Error`),
	regexp.MustCompile(`(?i)Bluetooth.*firmware`),
	regexp.MustCompile(`(?i)USB.*descriptor.*error`),
	regexp.MustCompile(`(?i)usb.*device descriptor read`),
	regexp.MustCompile(`(?i)firmware.*failed`),
	regexp.MustCompile(`(?i)GPU.*firmware`),
	regexp.MustCompile(`(?i)amdgpu.*error`),
	regexp.MustCompile(`(?i)i915.*error`),
	regexp.MustCompile(`(?i)nouveau.*error`),
	regexp.MustCompile(`(?i)Can't lookup blockdev`),
	regexp.MustCompile(`(?i)snapd.*Can't lookup blockdev`),
}

func compilePatterns(patterns []string) []*regexp.Regexp {
	var regexps []*regexp.Regexp
	for _, p := range patterns {
		re, err := regexp.Compile("(?i)" + p)
		if err == nil {
			regexps = append(regexps, re)
		} else {
			re, err = regexp.Compile("(?i)" + regexp.QuoteMeta(p))
			if err == nil {
				regexps = append(regexps, re)
			}
		}
	}
	return regexps
}

// CheckKernel checks for kernel errors in the current boot.
func CheckKernel(runner system.CommandRunner, config model.Config) model.Finding {
	finding := model.Finding{
		ID:           "kernel.errors.current_boot",
		Title:        "Kernel errors check",
		CheckCommand: "journalctl -k -p 3 -b --no-pager",
	}

	outputBytes, err := runner.Run("journalctl", "-k", "-p", "3", "-b", "--no-pager")
	if err != nil {
		finding.Severity = model.SeveritySkipped
		finding.Title = "Kernel error check skipped"
		finding.Summary = "Kernel journal could not be read."
		finding.Details = []string{fmt.Sprintf("Error: %v", err)}
		return finding
	}

	output := string(outputBytes)
	rawLines := strings.Split(output, "\n")

	ignoreRegexps := compilePatterns(config.Kernel.IgnorePatterns)
	downgradeRegexps := compilePatterns(config.Kernel.DowngradePatterns)

	var criticalLines []string
	var warningLines []string
	var unknownLines []string
	var totalLines int

	for _, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// 1. Check if line matches any ignored pattern
		ignored := false
		for _, re := range ignoreRegexps {
			if re.MatchString(trimmed) {
				ignored = true
				break
			}
		}
		if ignored {
			continue
		}

		totalLines++

		// 2. Check critical pattern matches
		isCritical := false
		for _, re := range criticalPatterns {
			if re.MatchString(trimmed) {
				isCritical = true
				break
			}
		}

		if isCritical {
			// Check if downgraded
			downgraded := false
			for _, re := range downgradeRegexps {
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

		// 3. Check warning pattern matches
		isWarning := false
		for _, re := range warningPatterns {
			if re.MatchString(trimmed) {
				isWarning = true
				break
			}
		}

		if isWarning {
			warningLines = append(warningLines, trimmed)
		} else {
			unknownLines = append(unknownLines, trimmed)
		}
	}

	if totalLines == 0 {
		finding.Severity = model.SeverityOK
		finding.Title = "No kernel errors found in current boot"
		finding.Summary = "No kernel priority-3 errors were found in this boot."
		return finding
	}

	// Determine severity based on logic
	if len(criticalLines) > 0 {
		finding.Severity = model.SeverityCritical
	} else if len(warningLines) > 0 {
		finding.Severity = model.SeverityWarning
	} else if totalLines > config.Kernel.UnknownErrorWarningCount {
		finding.Severity = model.SeverityWarning
	} else {
		finding.Severity = model.SeverityInfo
	}

	finding.Title = "Kernel errors found in current boot"
	finding.Summary = fmt.Sprintf("%d kernel error lines found in this boot.", totalLines)
	finding.Suggestion = "Inspect kernel errors for disk, driver, filesystem, or hardware problems."

	// Format details
	var details []string
	details = append(details, fmt.Sprintf("Total kernel error lines: %d", totalLines))
	details = append(details, fmt.Sprintf("Critical matches: %d", len(criticalLines)))
	details = append(details, fmt.Sprintf("Warning matches: %d", len(warningLines)))

	if len(criticalLines) > 0 {
		details = append(details, "Critical examples:")
		limit := 5
		if len(criticalLines) < limit {
			limit = len(criticalLines)
		}
		for i := 0; i < limit; i++ {
			details = append(details, fmt.Sprintf("  - %s", criticalLines[i]))
		}
	}

	if len(warningLines) > 0 {
		details = append(details, "Warning examples:")
		limit := 5
		if len(warningLines) < limit {
			limit = len(warningLines)
		}
		for i := 0; i < limit; i++ {
			details = append(details, fmt.Sprintf("  - %s", warningLines[i]))
		}
	}

	if len(unknownLines) > 0 {
		details = append(details, "Unknown examples:")
		limit := 5
		if len(unknownLines) < limit {
			limit = len(unknownLines)
		}
		for i := 0; i < limit; i++ {
			details = append(details, fmt.Sprintf("  - %s", unknownLines[i]))
		}
	}

	finding.Details = details
	return finding
}
