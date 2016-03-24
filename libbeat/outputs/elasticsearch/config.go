package elasticsearch

import (
	"time"

	"github.com/elastic/beats/libbeat/outputs"
)

type elasticsearchConfig struct {
	Protocol     string             `config:"protocol"`
	Path         string             `config:"path"`
	Params       map[string]string  `config:"parameters"`
	Username     string             `config:"username"`
	Password     string             `config:"password"`
	ProxyURL     string             `config:"proxy_url"`
	Index        string             `config:"index"`
	LoadBalance  bool               `config:"loadbalance"`
	TLS          *outputs.TLSConfig `config:"tls"`
	MaxRetries   int                `config:"max_retries"`
	Timeout      time.Duration      `config:"timeout"`
	SaveTopology bool               `config:"save_topology"`
	Template     Template           `config:"template"`
}

type Template struct {
	Name      string `config:"name"`
	Path      string `config:"path"`
	Overwrite bool   `config:"overwrite"`
}

const (
	defaultBulkSize = 50
)

var (
	defaultConfig = elasticsearchConfig{
		Protocol:    "",
		Path:        "",
		ProxyURL:    "",
		Params:      nil,
		Username:    "",
		Password:    "",
		Timeout:     90 * time.Second,
		MaxRetries:  3,
		TLS:         nil,
		LoadBalance: true,
	}
)
