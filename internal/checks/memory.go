package checks

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"faultradar/internal/config"
	"faultradar/internal/model"
	"faultradar/internal/system"
)

// CheckMemory checks memory and swap status using the injected FileSystem.
func CheckMemory(sysFS system.FileSystem, cfg config.Config) []model.Finding {
	checkCmd := "free -h"

	data, err := sysFS.ReadFile("/proc/meminfo")
	if err != nil {
		return []model.Finding{{
			ID:           "memory.unavailable",
			Severity:     model.SeveritySkipped,
			Title:        "Memory check skipped",
			Summary:      "/proc/meminfo could not be read.",
			CheckCommand: checkCmd,
			Details:      []string{fmt.Sprintf("Read error: %v", err)},
		}}
	}

	meminfo, err := parseMeminfoBytes(data)
	if err != nil {
		return []model.Finding{{
			ID:           "memory.unavailable",
			Severity:     model.SeveritySkipped,
			Title:        "Memory check skipped",
			Summary:      "/proc/meminfo could not be parsed.",
			CheckCommand: checkCmd,
			Details:      []string{fmt.Sprintf("Parse error: %v", err)},
		}}
	}

	required := []string{"MemTotal", "MemAvailable", "SwapTotal", "SwapFree"}
	for _, req := range required {
		if _, ok := meminfo[req]; !ok {
			return []model.Finding{{
				ID:           "memory.unavailable",
				Severity:     model.SeveritySkipped,
				Title:        "Memory check skipped",
				Summary:      fmt.Sprintf("Required memory stat %s was not found in /proc/meminfo.", req),
				CheckCommand: checkCmd,
			}}
		}
	}

	memTotal := meminfo["MemTotal"]
	memAvailable := meminfo["MemAvailable"]
	swapTotal := meminfo["SwapTotal"]
	swapFree := meminfo["SwapFree"]

	if memTotal <= 0 {
		return []model.Finding{{
			ID:           "memory.unavailable",
			Severity:     model.SeveritySkipped,
			Title:        "Memory check skipped",
			Summary:      "MemTotal is zero or negative.",
			CheckCommand: checkCmd,
		}}
	}

	availPercent := int((memAvailable * 100) / memTotal)
	warningThreshold := cfg.Memory.WarningAvailablePercent
	criticalThreshold := cfg.Memory.CriticalAvailablePercent

	details := []string{
		fmt.Sprintf("Total Memory: %d MB", memTotal/1024),
		fmt.Sprintf("Available Memory: %d MB", memAvailable/1024),
		fmt.Sprintf("Total Swap: %d MB", swapTotal/1024),
		fmt.Sprintf("Free Swap: %d MB", swapFree/1024),
	}

	var findings []model.Finding

	if availPercent <= criticalThreshold {
		findings = append(findings, model.Finding{
			ID:           "memory.critical",
			Severity:     model.SeverityCritical,
			Title:        "Memory available is critically low",
			Summary:      fmt.Sprintf("Available memory is %d%% of total (threshold: %d%%).", availPercent, criticalThreshold),
			Suggestion:   "Close unused applications or add RAM to the system.",
			CheckCommand: checkCmd,
			Details:      details,
		})
	} else if availPercent <= warningThreshold {
		findings = append(findings, model.Finding{
			ID:           "memory.low",
			Severity:     model.SeverityWarning,
			Title:        "Memory available is low",
			Summary:      fmt.Sprintf("Available memory is %d%% of total (threshold: %d%%).", availPercent, warningThreshold),
			Suggestion:   "Close unused applications or add memory.",
			CheckCommand: checkCmd,
			Details:      details,
		})
	}

	if swapTotal == 0 {
		findings = append(findings, model.Finding{
			ID:           "memory.no_swap",
			Severity:     model.SeverityWarning,
			Title:        "No swap detected",
			Summary:      "This system has no swap configured.",
			Suggestion:   "Under heavy memory pressure, the desktop may freeze.",
			CheckCommand: checkCmd,
			Details:      details,
		})
	}

	if len(findings) == 0 {
		findings = append(findings, model.Finding{
			ID:           "memory.ok",
			Severity:     model.SeverityOK,
			Title:        "Memory status looks normal",
			Summary:      fmt.Sprintf("Available memory is %d%% of total.", availPercent),
			CheckCommand: checkCmd,
			Details:      details,
		})
	}

	return findings
}

func parseMeminfoBytes(data []byte) (map[string]int64, error) {
	res := make(map[string]int64)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		valStr := strings.TrimSpace(parts[1])
		valStr = strings.TrimSuffix(valStr, " kB")
		valStr = strings.TrimSpace(valStr)

		fields := strings.Fields(valStr)
		if len(fields) == 0 {
			continue
		}
		val, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			continue
		}
		res[key] = val
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return res, nil
}
