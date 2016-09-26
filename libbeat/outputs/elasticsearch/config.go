package elasticsearch

import (
	"time"

	"github.com/elastic/beats/libbeat/outputs"
)

type elasticsearchConfig struct {
	Protocol         string             `config:"protocol"`
	Path             string             `config:"path"`
	Params           map[string]string  `config:"parameters"`
	Username         string             `config:"username"`
	Password         string             `config:"password"`
	ProxyURL         string             `config:"proxy_url"`
	LoadBalance      bool               `config:"loadbalance"`
	CompressionLevel int                `config:"compression_level" validate:"min=0, max=9"`
	TLS              *outputs.TLSConfig `config:"ssl"`
	MaxRetries       int                `config:"max_retries"`
	Timeout          time.Duration      `config:"timeout"`
	SaveTopology     bool               `config:"save_topology"`
	Template         Template           `config:"template"`
}

type Template struct {
	Enabled   bool             `config:"enabled"`
	Name      string           `config:"name"`
	Path      string           `config:"path"`
	Overwrite bool             `config:"overwrite"`
	Versions  TemplateVersions `config:"versions"`
}

type TemplateVersions struct {
	Es2x TemplateVersion `config:"2x"`
}

type TemplateVersion struct {
	Enabled bool   `config:"enabled"`
	Path    string `config:"path"`
}

const (
	defaultBulkSize = 50
)

var (
	defaultConfig = elasticsearchConfig{
		Protocol:         "",
		Path:             "",
		ProxyURL:         "",
		Params:           nil,
		Username:         "",
		Password:         "",
		Timeout:          90 * time.Second,
		MaxRetries:       3,
		CompressionLevel: 0,
		TLS:              nil,
		LoadBalance:      true,
		Template: Template{
			Enabled:  true,
			Versions: TemplateVersions{Es2x: TemplateVersion{Enabled: true}},
		},
	}
)

func (c *elasticsearchConfig) Validate() error {
	if c.ProxyURL != "" {
		if _, err := parseProxyURL(c.ProxyURL); err != nil {
			return err
		}
	}

	return nil
}
