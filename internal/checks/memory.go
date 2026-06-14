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
func CheckMemory(sysFS system.FileSystem, cfg config.Config) model.Finding {
	finding := model.Finding{
		ID:           "memory.available",
		Title:        "Memory available check",
		CheckCommand: "free -h",
	}

	data, err := sysFS.ReadFile("/proc/meminfo")
	if err != nil {
		finding.Severity = model.SeveritySkipped
		finding.Title = "Memory check skipped"
		finding.Summary = "/proc/meminfo could not be read."
		finding.Details = []string{fmt.Sprintf("Read error: %v", err)}
		return finding
	}

	meminfo, err := parseMeminfoBytes(data)
	if err != nil {
		finding.Severity = model.SeveritySkipped
		finding.Title = "Memory check skipped"
		finding.Summary = "/proc/meminfo could not be parsed."
		finding.Details = []string{fmt.Sprintf("Parse error: %v", err)}
		return finding
	}

	required := []string{"MemTotal", "MemAvailable", "SwapTotal", "SwapFree"}
	for _, req := range required {
		if _, ok := meminfo[req]; !ok {
			finding.Severity = model.SeveritySkipped
			finding.Title = "Memory check skipped"
			finding.Summary = fmt.Sprintf("Required memory stat %s was not found in /proc/meminfo.", req)
			return finding
		}
	}

	memTotal := meminfo["MemTotal"]
	memAvailable := meminfo["MemAvailable"]
	swapTotal := meminfo["SwapTotal"]
	swapFree := meminfo["SwapFree"]

	if memTotal <= 0 {
		finding.Severity = model.SeveritySkipped
		finding.Title = "Memory check skipped"
		finding.Summary = "MemTotal is zero or negative."
		return finding
	}

	availPercent := int((memAvailable * 100) / memTotal)

	warningThreshold := cfg.Memory.AvailableWarningPercent
	criticalThreshold := cfg.Memory.AvailableCriticalPercent

	finding.Details = []string{
		fmt.Sprintf("Total Memory: %d MB", memTotal/1024),
		fmt.Sprintf("Available Memory: %d MB (%d%%)", memAvailable/1024, availPercent),
		fmt.Sprintf("Total Swap: %d MB", swapTotal/1024),
		fmt.Sprintf("Free Swap: %d MB", swapFree/1024),
	}

	if availPercent <= criticalThreshold {
		finding.Severity = model.SeverityCritical
		finding.Title = "Memory available is critically low"
		finding.Summary = fmt.Sprintf("Available memory is %d%% (threshold: %d%%).", availPercent, criticalThreshold)
		finding.Suggestion = "Close some unused applications or add RAM to the system."
	} else if availPercent <= warningThreshold {
		finding.Severity = model.SeverityWarning
		finding.Title = "Memory available is low"
		finding.Summary = fmt.Sprintf("Available memory is %d%% (threshold: %d%%).", availPercent, warningThreshold)
		finding.Suggestion = "Close some unused applications or add memory."
	} else if swapTotal == 0 && availPercent <= warningThreshold+10 {
		finding.Severity = model.SeverityWarning
		finding.Title = "No swap configured and memory is low"
		finding.Summary = fmt.Sprintf("No swap exists and available memory is %d%%.", availPercent)
		finding.Suggestion = "Configure a swap file or partition to prevent OOM freezes under memory pressure."
	} else {
		finding.Severity = model.SeverityOK
		finding.Title = "Memory status looks normal"
		finding.Summary = fmt.Sprintf("Available memory is %d%%.", availPercent)
	}

	return finding
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
