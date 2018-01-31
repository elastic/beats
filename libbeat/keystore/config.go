package keystore

// Config Define keystore configurable options
type Config struct {
	Path string `config:"path"`
}

var defaultConfig = Config{
	Path: "",
}
