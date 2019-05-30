package remote_write

import "github.com/elastic/beats/libbeat/common/transport/tlscommon"

type Config struct {
	Host string                  `config:"host"`
	Port int                     `config:"port"`
	TLS  *tlscommon.ServerConfig `config:"ssl"`
}

func defaultConfig() Config {
	return Config{
		Host: "localhost",
		Port: 9201,
	}
}
