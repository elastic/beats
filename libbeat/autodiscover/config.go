package autodiscover

import (
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
)

// Config settings for Autodiscover
type Config struct {
	Providers []*common.Config `config:"providers"`
}

// ProviderConfig settings
type ProviderConfig struct {
	Type      string                  `config:"type"`
	Builders  []*common.Config        `config:"builders"`
	Templates template.MapperSettings `config:"templates"`
}

// BuilderConfig settings
type BuilderConfig struct {
	Type string `config:"type"`
}
