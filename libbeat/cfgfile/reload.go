package cfgfile

import "time"

var (
	DefaultReloadConfig = ReloadConfig{
		Period:  10 * time.Second,
		Enabled: false,
	}
)

type ReloadConfig struct {
	// If path is a relative path, it is relative to the ${path.config}
	Path    string        `config:"path"`
	Period  time.Duration `config:"period"`
	Enabled bool          `config:"enabled"`
}
