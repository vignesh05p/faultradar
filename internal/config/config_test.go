package config

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Disk.WarningPercent != 90 {
		t.Errorf("expected disk warning_percent 90, got %d", cfg.Disk.WarningPercent)
	}
	if cfg.Disk.CriticalPercent != 97 {
		t.Errorf("expected disk critical_percent 97, got %d", cfg.Disk.CriticalPercent)
	}
	if cfg.Logs.WarningMB != 1024 {
		t.Errorf("expected logs warning_mb 1024, got %d", cfg.Logs.WarningMB)
	}
	if cfg.Logs.CriticalMB != 5120 {
		t.Errorf("expected logs critical_mb 5120, got %d", cfg.Logs.CriticalMB)
	}
	if cfg.Memory.WarningAvailablePercent != 10 {
		t.Errorf("expected memory warning_available_percent 10, got %d", cfg.Memory.WarningAvailablePercent)
	}
	if cfg.Memory.CriticalAvailablePercent != 5 {
		t.Errorf("expected memory critical_available_percent 5, got %d", cfg.Memory.CriticalAvailablePercent)
	}
	if len(cfg.Systemd.ImportantUnits) == 0 {
		t.Error("expected default important units to be populated")
	}
}
