package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"faultradar/internal/checks"
	"faultradar/internal/model"
	"faultradar/internal/system"
)

type Doctor struct {
	Config model.Config
	Runner system.CommandRunner
	FS     system.FileSystem
}

func (d Doctor) Run() []model.Finding {
	var findings []model.Finding

	findings = append(findings, checks.CheckDisk(d.FS, d.Config))
	findings = append(findings, checks.CheckLogs(d.FS, d.Config))
	findings = append(findings, checks.CheckSystemd(d.Runner, d.Config))
	findings = append(findings, checks.CheckKernel(d.Runner, d.Config))
	findings = append(findings, checks.CheckMemory(d.FS, d.Config))

	return findings
}

// LoadConfig searches for the configuration file in standard locations
// and decodes it, fallback to default config if none is found.
// It returns the config and optionally a list of warning findings if loading had non-fatal errors.
func LoadConfig(fs system.FileSystem) (model.Config, []model.Finding) {
	defaultConf := model.DefaultConfig()
	var findings []model.Finding

	var paths []string

	homeDir, err := os.UserHomeDir()
	if err == nil {
		paths = append(paths, filepath.Join(homeDir, ".config", "faultradar", "config.json"))
	}
	paths = append(paths, "/etc/faultradar/config.json")

	var foundPath string
	var configData []byte

	for _, path := range paths {
		data, err := fs.ReadFile(path)
		if err == nil {
			foundPath = path
			configData = data
			break
		}
	}

	if foundPath == "" {
		return defaultConf, nil
	}

	userConf := defaultConf
	err = json.Unmarshal(configData, &userConf)
	if err != nil {
		findings = append(findings, model.Finding{
			ID:       "config.load",
			Severity: model.SeverityWarning,
			Title:    "Invalid configuration file format",
			Summary:  fmt.Sprintf("Failed to parse configuration file at %s.", foundPath),
			Details:  []string{fmt.Sprintf("Error: %v", err)},
		})
		return defaultConf, findings
	}

	return userConf, findings
}

// ExitCode returns the appropriate exit code based on the worst severity in findings.
func ExitCode(findings []model.Finding) int {
	hasCritical := false
	hasWarning := false

	for _, f := range findings {
		if f.Severity == model.SeverityCritical {
			hasCritical = true
		} else if f.Severity == model.SeverityWarning {
			hasWarning = true
		}
	}

	if hasCritical {
		return 2
	}
	if hasWarning {
		return 1
	}
	return 0
}
