package checks

import (
	"fmt"
	"strings"

	"faultradar/internal/model"
	"faultradar/internal/system"
)

// CheckKernel checks for kernel errors in the current boot.
func CheckKernel(runner system.CommandRunner, config model.Config) model.Finding {
	finding := model.Finding{
		ID:           "kernel.errors.current_boot",
		Title:        "Kernel errors check",
		CheckCommand: "journalctl -k -p 3 -b --no-pager",
	}

	outputBytes, err := runner.Run("journalctl", "-k", "-p", "3", "-b", "--no-pager")
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "not found") || strings.Contains(errStr, "no such file") || strings.Contains(errStr, "executable file not found") {
			finding.Severity = model.SeveritySkipped
			finding.Title = "Kernel error diagnostics unavailable"
			finding.Summary = "journalctl command was not found on this system."
			finding.Details = []string{fmt.Sprintf("Error: %v", err)}
			return finding
		}
		finding.Severity = model.SeveritySkipped
		finding.Title = "Kernel error check skipped"
		finding.Summary = "Failed to run journalctl command."
		finding.Details = []string{fmt.Sprintf("Run error: %v", err)}
		return finding
	}

	output := string(outputBytes)
	rawLines := strings.Split(output, "\n")
	var errorLines []string
	for _, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			errorLines = append(errorLines, trimmed)
		}
	}

	count := len(errorLines)
	warningThreshold := config.Kernel.MaxErrorLinesWarning
	criticalThreshold := config.Kernel.MaxErrorLinesCritical

	var sampleLines []string
	limit := 5
	if count < limit {
		limit = count
	}
	for i := 0; i < limit; i++ {
		sampleLines = append(sampleLines, errorLines[i])
	}

	if count >= criticalThreshold {
		finding.Severity = model.SeverityCritical
		finding.Title = "Kernel errors found in current boot"
		finding.Summary = fmt.Sprintf("%d kernel error lines found in this boot.", count)
		finding.Suggestion = "Inspect kernel errors for disk, driver, filesystem, or hardware problems."
		finding.Details = append([]string{fmt.Sprintf("First %d kernel errors:", limit)}, sampleLines...)
	} else if count >= warningThreshold {
		finding.Severity = model.SeverityWarning
		finding.Title = "Kernel errors found in current boot"
		finding.Summary = fmt.Sprintf("%d kernel error lines found in this boot.", count)
		finding.Suggestion = "Inspect kernel errors for disk, driver, filesystem, or hardware problems."
		finding.Details = append([]string{fmt.Sprintf("First %d kernel errors:", limit)}, sampleLines...)
	} else {
		finding.Severity = model.SeverityOK
		finding.Title = "Kernel logs look normal"
		finding.Summary = "No significant kernel errors found in current boot."
		if count > 0 {
			finding.Details = append([]string{fmt.Sprintf("First %d kernel errors:", limit)}, sampleLines...)
		}
	}

	return finding
}
