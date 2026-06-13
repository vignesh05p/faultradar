package model

type Config struct {
	Disk struct {
		RootWarningPercent  int `json:"root_warning_percent"`
		RootCriticalPercent int `json:"root_critical_percent"`
	} `json:"disk"`

	Logs struct {
		VarLogWarningMB  int64 `json:"varlog_warning_mb"`
		VarLogCriticalMB int64 `json:"varlog_critical_mb"`
	} `json:"logs"`

	Kernel struct {
		UnknownErrorWarningCount int      `json:"unknown_error_warning_count"`
		DowngradePatterns        []string `json:"downgrade_patterns"`
		IgnorePatterns           []string `json:"ignore_patterns"`
	} `json:"kernel"`

	Systemd struct {
		IgnoreUnits        []string `json:"ignore_units"`
		IgnoreUnitPatterns []string `json:"ignore_unit_patterns"`
		ImportantUnits     []string `json:"important_units"`
	} `json:"systemd"`

	Memory struct {
		AvailableWarningPercent  int `json:"available_warning_percent"`
		AvailableCriticalPercent int `json:"available_critical_percent"`
	} `json:"memory"`
}

// DefaultConfig returns the default configuration for FaultRadar.
func DefaultConfig() Config {
	var c Config
	c.Disk.RootWarningPercent = 85
	c.Disk.RootCriticalPercent = 95

	c.Logs.VarLogWarningMB = 1024
	c.Logs.VarLogCriticalMB = 5120

	c.Kernel.UnknownErrorWarningCount = 10
	c.Kernel.IgnorePatterns = []string{}
	c.Kernel.DowngradePatterns = []string{}

	c.Systemd.IgnoreUnits = []string{}
	c.Systemd.IgnoreUnitPatterns = []string{}
	c.Systemd.ImportantUnits = []string{
		"mysql.service",
		"postgresql.service",
		"docker.service",
		"containerd.service",
		"ssh.service",
		"sshd.service",
		"NetworkManager.service",
		"systemd-resolved.service",
		"systemd-journald.service",
	}

	c.Memory.AvailableWarningPercent = 15
	c.Memory.AvailableCriticalPercent = 5
	return c
}
