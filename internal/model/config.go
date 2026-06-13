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
		MaxErrorLinesWarning  int `json:"max_error_lines_warning"`
		MaxErrorLinesCritical int `json:"max_error_lines_critical"`
	} `json:"kernel"`

	Systemd struct {
		IgnoreUnits []string `json:"ignore_units"`
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
	c.Kernel.MaxErrorLinesWarning = 5
	c.Kernel.MaxErrorLinesCritical = 25
	c.Memory.AvailableWarningPercent = 15
	c.Memory.AvailableCriticalPercent = 5
	c.Systemd.IgnoreUnits = []string{}
	return c
}
