package module

import "time"

type ReloaderConfig struct {
	// If path is a relative path, it is relative to the ${path.config}
	Path    string        `config:"path"`
	Period  time.Duration `config:"period"`
	Enabled bool          `config:"enabled"`
}

var (
	DefaultReloaderConfig = ReloaderConfig{
		Period:  10 * time.Second,
		Enabled: false,
	}
)
