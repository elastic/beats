package autodiscover

import "github.com/elastic/beats/libbeat/common"

// Config settings for Autodiscover
type Config struct {
	Providers []*common.Config `config:"providers"`
}

// ProviderConfig settings
type ProviderConfig struct {
	Type string `config:"type"`
}
