package server

import (
	"errors"

	"github.com/elastic/beats/libbeat/common"
)

type httpServerConfig struct {
	Paths       []pathConfig `config:"paths"`
	DefaultPath pathConfig   `config:"default_path"`
}

type pathConfig struct {
	Path      string        `config:"path"`
	Fields    common.MapStr `config:"fields"`
	Namespace string        `config:"namespace"`
}

func defaultHttpServerConfig() httpServerConfig {
	return httpServerConfig{
		DefaultPath: pathConfig{
			Path:      "/",
			Namespace: "http",
		},
	}
}

func (p pathConfig) Validate() error {
	if p.Namespace == "" {
		return errors.New("`namespace` can not be empty in path configuration")
	}

	if p.Path == "" {
		return errors.New("`path` can not be empty in path configuration")
	}

	return nil
}
