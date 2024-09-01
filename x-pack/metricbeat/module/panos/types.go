package panos

const (
	ModuleName = "panos"
)

type Config struct {
	HostIp    string `config:"host_ip"`
	ApiKey    string `config:"apiKey"`
	DebugMode string `config:"apiDebugMode"`
}
