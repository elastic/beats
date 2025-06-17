package ntp

import (
	"fmt"
	"time"
)

type config struct {
	Host    string        `config:"host"`
	Timeout time.Duration `config:"timeout"`
	Version int           `config:"version"`
}

func defaultConfig() config {
	return config{
		Timeout: 5 * time.Second,
		Version: 4,
	}
}

func validateConfig(cfg *config) error {
	if cfg.Host == "" {
		return fmt.Errorf("NTP host must be set in config")
	}
	if cfg.Timeout <= 0 {
		return fmt.Errorf("invalid NTP timeout: %s", cfg.Timeout.String())
	}
	if cfg.Version != 3 && cfg.Version != 4 {
		return fmt.Errorf("NTP version must be 3 or 4")
	}
	return nil
}
