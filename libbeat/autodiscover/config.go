package autodiscover

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

// Config settings for Autodiscover
type Config struct {
	Providers []*common.Config `config:"providers"`
}

// ProviderConfig settings
type ProviderConfig struct {
	Type string `config:"type"`
}

// BuilderConfig settings
type BuilderConfig struct {
	Type string `config:"type"`
}

// AppenderConfig settings
type AppenderConfig struct {
	Type            string                      `config:"type"`
	ConditionConfig *processors.ConditionConfig `config:"condition"`
}
