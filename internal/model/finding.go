package model

type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
	SeveritySkipped  Severity = "skipped"
	SeverityOK       Severity = "ok"
)

type Finding struct {
	ID           string   `json:"id"`
	Severity     Severity `json:"severity"`
	Title        string   `json:"title"`
	Summary      string   `json:"summary"`
	Suggestion   string   `json:"suggestion,omitempty"`
	CheckCommand string   `json:"check_command,omitempty"`
	Details      []string `json:"details,omitempty"`
}
