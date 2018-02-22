package logp

// Config contains the configuration options for the logger. To create a Config
// from a common.Config use logp/config.Build.
type Config struct {
	Beat      string   `config:",ignore"`   // Name of the Beat (for default file name).
	JSON      bool     `config:"json"`      // Write logs as JSON.
	Level     Level    `config:"level"`     // Logging level (error, warning, info, debug).
	Selectors []string `config:"selectors"` // Selectors for debug level logging.

	toObserver  bool
	toIODiscard bool
	ToStderr    bool `config:"to_stderr"`
	ToSyslog    bool `config:"to_syslog"`
	ToFiles     bool `config:"to_files"`
	ToEventLog  bool `config:"to_eventlog"`

	Files FileConfig `config:"files"`

	addCaller   bool // Adds package and line number info to messages.
	development bool // Controls how DPanic behaves.
}

// FileConfig contains the configuration options for the file output.
type FileConfig struct {
	Path        string `config:"path"`
	Name        string `config:"name"`
	MaxSize     uint   `config:"rotateeverybytes" validate:"min=1"`
	MaxBackups  uint   `config:"keepfiles" validate:"max=1024"`
	Permissions uint32 `config:"permissions"`
}

var defaultConfig = Config{
	Level:   InfoLevel,
	ToFiles: true,
	Files: FileConfig{
		MaxSize:     10 * 1024 * 1024,
		Permissions: 0600,
	},
	addCaller: true,
}

// DefaultConfig returns the default config options.
func DefaultConfig() Config {
	return defaultConfig
}
