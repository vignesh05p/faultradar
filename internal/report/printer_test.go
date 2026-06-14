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
			ID:           "memory.low",
			Severity:     model.SeverityWarning,
			Title:        "Memory available is low",
			Summary:      "Memory is 8% available.",
			Suggestion:   "Close apps.",
			CheckCommand: "free -h",
		},
		{
			ID:       "logs.varlog.size",
			Severity: model.SeverityOK,
			Title:    "Log storage looks normal",
			Summary:  "Total log size is 100 MB.",
		},
		{
			ID:       "some.info.id",
			Severity: model.SeverityInfo,
			Title:    "System info diagnostic",
			Summary:  "Normal system metadata.",
		},
		{
			ID:       "some.skipped.id",
			Severity: model.SeveritySkipped,
			Title:    "Skipped check info",
			Summary:  "Failed to run check.",
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

		if len(parsed) != 5 {
			t.Errorf("expected 5 findings, got %d", len(parsed))
		}
	})

	t.Run("JSON is valid and pure", func(t *testing.T) {
		var buf bytes.Buffer
		err := PrintJSON(&buf, findings)
		if err != nil {
			t.Fatalf("PrintJSON failed: %v", err)
		}
		trimmed := strings.TrimSpace(buf.String())
		if !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
			t.Errorf("JSON output has extra text or invalid root container: %s", trimmed)
		}
	})

	t.Run("human severity ordering", func(t *testing.T) {
		var buf bytes.Buffer
		PrintHuman(&buf, "1.0.0", findings)
		out := buf.String()

		if !strings.Contains(out, "FaultRadar v1.0.0") {
			t.Errorf("expected version header")
		}

		critIdx := strings.Index(out, "CRITICAL")
		warnIdx := strings.Index(out, "WARNING")
		infoIdx := strings.Index(out, "INFO")
		skipIdx := strings.Index(out, "SKIPPED")
		okIdx := strings.Index(out, "OK")

		if critIdx == -1 || warnIdx == -1 || infoIdx == -1 || skipIdx == -1 || okIdx == -1 {
			t.Errorf("expected all severity headers to be present")
		}

		if !(critIdx < warnIdx && warnIdx < infoIdx && infoIdx < skipIdx && skipIdx < okIdx) {
			t.Errorf("severity groups printed in incorrect order")
		}
	})

	t.Run("no empty sections", func(t *testing.T) {
		var buf bytes.Buffer
		PrintHuman(&buf, "1.0.0", []model.Finding{
			{ID: "disk.root.usage", Severity: model.SeverityOK, Title: "Root disk ok", Summary: "42% used."},
		})
		out := buf.String()
		if strings.Contains(out, "CRITICAL") || strings.Contains(out, "WARNING") {
			t.Errorf("expected no empty severity sections")
		}
	})

	t.Run("skipped group appears if skipped finding exists", func(t *testing.T) {
		var buf bytes.Buffer
		PrintHuman(&buf, "1.0.0", findings)
		out := buf.String()

		if !strings.Contains(out, "SKIPPED") {
			t.Errorf("expected output to contain SKIPPED group header")
		}
		if !strings.Contains(out, "Skipped check info") {
			t.Errorf("expected output to contain skipped check details")
		}
	})
}
