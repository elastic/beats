package kibana

import (
	"time"

	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
)

type kibanaConfig struct {
	Protocol string            `config:"protocol"`
	Host     string            `config:"host"`
	Path     string            `config:"path"`
	Username string            `config:"username"`
	Password string            `config:"password"`
	TLS      *tlscommon.Config `config:"ssl"`
	Timeout  time.Duration     `config:"timeout"`
}

var (
	defaultKibanaConfig = kibanaConfig{
		Protocol: "http",
		Host:     "localhost:5601",
		Path:     "",
		Username: "",
		Password: "",
		Timeout:  90 * time.Second,
		TLS:      nil,
	}
)
