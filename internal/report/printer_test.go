package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"faultradar/internal/model"
)

func TestPrinter(t *testing.T) {
	findings := []model.Finding{
		{
			ID:           "disk.root.usage",
			Severity:     model.SeverityCritical,
			Title:        "Root disk is almost full",
			Summary:      "Root filesystem is 97% used.",
			Suggestion:   "Free disk space.",
			CheckCommand: "df -h /",
		},
		{
			ID:           "memory.available",
			Severity:     model.SeverityWarning,
			Title:        "Memory available is low",
			Summary:      "Memory is 12% available.",
			Suggestion:   "Close apps.",
			CheckCommand: "free -h",
		},
		{
			ID:       "logs.varlog.size",
			Severity: model.SeverityOK,
			Title:    "Log files size looks normal",
			Summary:  "Total log size is 100 MB.",
		},
	}

	t.Run("JSON output parses", func(t *testing.T) {
		var buf bytes.Buffer
		err := PrintJSON(&buf, findings)
		if err != nil {
			t.Fatalf("PrintJSON failed: %v", err)
		}

		var parsed []model.Finding
		err = json.Unmarshal(buf.Bytes(), &parsed)
		if err != nil {
			t.Fatalf("Failed to parse printed JSON: %v", err)
		}

		if len(parsed) != 3 {
			t.Errorf("expected 3 findings, got %d", len(parsed))
		}
		if parsed[0].ID != "disk.root.usage" {
			t.Errorf("expected ID %q, got %q", "disk.root.usage", parsed[0].ID)
		}
	})

	t.Run("human output groups severities", func(t *testing.T) {
		var buf bytes.Buffer
		PrintHuman(&buf, "1.0.0", findings)
		out := buf.String()

		if !strings.Contains(out, "FaultRadar v1.0.0") {
			t.Errorf("expected output to contain version header")
		}
		if !strings.Contains(out, "CRITICAL") {
			t.Errorf("expected output to contain CRITICAL group")
		}
		if !strings.Contains(out, "WARNING") {
			t.Errorf("expected output to contain WARNING group")
		}
		if !strings.Contains(out, "OK") {
			t.Errorf("expected output to contain OK group")
		}

		critIdx := strings.Index(out, "CRITICAL")
		warnIdx := strings.Index(out, "WARNING")
		okIdx := strings.Index(out, "OK")

		if critIdx > warnIdx || warnIdx > okIdx {
			t.Errorf("expected groups to be ordered: CRITICAL, WARNING, OK")
		}
	})
}
