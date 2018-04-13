package kubernetes

import (
	"time"

	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
)

// Config for kubernetes autodiscover provider
type Config struct {
	InCluster      bool          `config:"in_cluster"`
	KubeConfig     string        `config:"kube_config"`
	Host           string        `config:"host"`
	Namespace      string        `config:"namespace"`
	SyncPeriod     time.Duration `config:"sync_period"`
	CleanupTimeout time.Duration `config:"cleanup_timeout"`

	IncludeLabels      []string `config:"include_labels"`
	ExcludeLabels      []string `config:"exclude_labels"`
	IncludeAnnotations []string `config:"include_annotations"`

	Prefix       string                  `config:"prefix"`
	HintsEnabled bool                    `config:"hints.enabled"`
	Builders     []*common.Config        `config:"builders"`
	Appenders    []*common.Config        `config:"appenders"`
	Templates    template.MapperSettings `config:"templates"`
}

func defaultConfig() *Config {
	return &Config{
		InCluster:      true,
		SyncPeriod:     1 * time.Second,
		CleanupTimeout: 60 * time.Second,
		Prefix:         "co.elastic",
	}
}

// Validate ensures correctness of config
func (c *Config) Validate() {
	// Make sure that prefix doesn't ends with a '.'
	if c.Prefix[len(c.Prefix)-1] == '.' && c.Prefix != "." {
		c.Prefix = c.Prefix[:len(c.Prefix)-2]
	}
}
