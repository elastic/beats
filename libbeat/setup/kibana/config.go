package kibana

type kibanaConfig struct {
	Protocol string `config:"protocol"`
	Host     string `config:"host"`
	Path     string `config:"path"`
}

var (
	defaultKibanaConfig = kibanaConfig{
		Protocol: "http",
		Host:     "",
		Path:     "",
	}
)
