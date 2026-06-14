package config

type Config struct {
	Disk struct {
		WarningPercent  int `json:"warning_percent"`
		CriticalPercent int `json:"critical_percent"`
	} `json:"disk"`

	Logs struct {
		WarningMB  int64 `json:"warning_mb"`
		CriticalMB int64 `json:"critical_mb"`
	} `json:"logs"`

	Kernel struct {
		IgnorePatterns    []string `json:"ignore_patterns"`
		DowngradePatterns []string `json:"downgrade_patterns"`
	} `json:"kernel"`

	Systemd struct {
		IgnoreUnits        []string `json:"ignore_units"`
		IgnoreUnitPatterns []string `json:"ignore_unit_patterns"`
		ImportantUnits     []string `json:"important_units"`
	} `json:"systemd"`

	Memory struct {
		WarningAvailablePercent  int `json:"warning_available_percent"`
		CriticalAvailablePercent int `json:"critical_available_percent"`
	} `json:"memory"`
}

// DefaultConfig returns the default configuration for FaultRadar.
func DefaultConfig() Config {
	var c Config
	c.Disk.WarningPercent = 90
	c.Disk.CriticalPercent = 97

	c.Logs.WarningMB = 1024
	c.Logs.CriticalMB = 5120

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
		"gdm.service",
		"sddm.service",
		"lightdm.service",
		"display-manager.service",
	}

	c.Memory.WarningAvailablePercent = 10
	c.Memory.CriticalAvailablePercent = 5
	return c
}
