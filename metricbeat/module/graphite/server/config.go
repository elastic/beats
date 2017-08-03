package server

import (
	"errors"
)

const (
	defaultDelimiter = "."
)

type GraphiteServerConfig struct {
	Protocol        string           `config:"protocol"`
	Templates       []TemplateConfig `config:"templates"`
	DefaultTemplate TemplateConfig   `config:"default_template"`
}

type TemplateConfig struct {
	Filter    string            `config:"filter"`
	Template  string            `config:"template"`
	Namespace string            `config:"namespace"`
	Delimiter string            `config:"delimiter"`
	Tags      map[string]string `config:"tags"`
}

func DefaultGraphiteCollectorConfig() GraphiteServerConfig {
	return GraphiteServerConfig{
		Protocol: "udp",
		DefaultTemplate: TemplateConfig{
			Filter:    "*",
			Template:  "metric*",
			Namespace: "graphite",
			Delimiter: ".",
		},
	}
}

func (c GraphiteServerConfig) Validate() error {
	if c.Protocol != "tcp" && c.Protocol != "udp" {
		return errors.New("`protocol` can only be tcp or udp")
	}
	return nil
}

func (t *TemplateConfig) Validate() error {
	if t.Namespace == "" {
		return errors.New("`namespace` can not be empty in template configuration")
	}

	if t.Filter == "" {
		return errors.New("`filter` can not be empty in template configuration")
	}

	if t.Template == "" {
		return errors.New("`template` can not be empty in template configuration")
	}

	if t.Delimiter == "" {
		t.Delimiter = defaultDelimiter
	}

	return nil
}
