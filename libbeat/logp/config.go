package logp

type Config struct {
	ToStderr  bool       `config:"to_stderr"`
	ToSyslog  bool       `config:"to_syslog"`
	ToFiles   bool       `config:"to_files"`
	Level     string     `config:"level"`
	Selectors []string   `config:"selectors"`
	JSON      bool       `config:"json"`
	Files     FileConfig `config:"files"`
}

type FileConfig struct {
	Path             string `config:"path"`
	Name             string `config:"name"`
	RotateEveryBytes uint64 `config:"rotateeverybytes"`
	KeepFiles        int    `config:"keepfiles"`
	Permissions      uint32 `config:"permissions"`
}

var DefaultConfig = Config{
	Level:   "info",
	ToFiles: true,
	Files: FileConfig{
		Name:             "beat.log",
		RotateEveryBytes: 10 * 1024 * 1024,
		KeepFiles:        7,
		Permissions:      0600,
	},
}
