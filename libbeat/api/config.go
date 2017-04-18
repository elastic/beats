package api

type Config struct {
	Enabled bool
	Host    string
	Port    int
}

var (
	DefaultConfig = Config{
		Enabled: false,
		Host:    "localhost",
		Port:    5066,
	}
)
