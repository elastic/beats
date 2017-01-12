package module

import "time"

// ReloaderConfig contains config options for the module Reloader.
type ReloaderConfig struct {
	// If path is a relative path, it is relative to the ${path.config}
	Path    string        `config:"path"`
	Period  time.Duration `config:"period"`
	Enabled bool          `config:"enabled"`
}

var (
	// DefaultReloaderConfig contains the default config options.
	DefaultReloaderConfig = ReloaderConfig{
		Period:  10 * time.Second,
		Enabled: false,
	}
)
