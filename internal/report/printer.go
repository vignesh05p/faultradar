package report

import (
	"encoding/json"
	"fmt"
	"io"

	"faultradar/internal/model"
)

// PrintHuman prints the findings in a friendly grouped human-readable format.
func PrintHuman(w io.Writer, version string, findings []model.Finding) {
	fmt.Fprintf(w, "FaultRadar v%s\n", version)

	// Group findings by severity
	groups := map[model.Severity][]model.Finding{
		model.SeverityCritical: {},
		model.SeverityWarning:  {},
		model.SeverityInfo:     {},
		model.SeveritySkipped:  {},
		model.SeverityOK:       {},
	}

	for _, f := range findings {
		groups[f.Severity] = append(groups[f.Severity], f)
	}

	severityOrder := []struct {
		sev  model.Severity
		name string
	}{
		{model.SeverityCritical, "CRITICAL"},
		{model.SeverityWarning, "WARNING"},
		{model.SeverityInfo, "INFO"},
		{model.SeveritySkipped, "SKIPPED"},
		{model.SeverityOK, "OK"},
	}

	counter := 1

	for _, order := range severityOrder {
		list := groups[order.sev]
		if len(list) == 0 {
			continue
		}

		fmt.Fprintln(w)
		fmt.Fprintln(w, order.name)

		for _, f := range list {
			fmt.Fprintln(w)
			printFinding(w, f, counter)
			counter++
		}
	}
}

// PrintJSON serializes the findings to w in JSON format.
func PrintJSON(w io.Writer, findings []model.Finding) error {
	if findings == nil {
		findings = []model.Finding{}
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(findings)
}

func printFinding(w io.Writer, f model.Finding, index int) {
	fmt.Fprintf(w, "[%d] %s\n", index, f.Title)

	if f.Summary != "" {
		fmt.Fprintf(w, "    %s\n", f.Summary)
	}

	if f.Suggestion != "" {
		fmt.Fprintf(w, "    Suggestion: %s\n", f.Suggestion)
	}

	if f.CheckCommand != "" {
		fmt.Fprintln(w, "    Check:")
		fmt.Fprintf(w, "      %s\n", f.CheckCommand)
	}

	if len(f.Details) > 0 {
		fmt.Fprintln(w, "    Details:")
		for _, detail := range f.Details {
			fmt.Fprintf(w, "      %s\n", detail)
		}
	}
}
