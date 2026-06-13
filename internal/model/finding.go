package model

type Severity string

const (
	SeverityOK       Severity = "ok"
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
	SeveritySkipped  Severity = "skipped"
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
