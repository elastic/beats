package server

import (
	"errors"

	"github.com/elastic/beats/libbeat/common"
)

type HttpServerConfig struct {
	Paths       []PathConfig `config:"server.paths"`
	DefaultPath PathConfig   `config:"server.default_path"`
}

type PathConfig struct {
	Path      string        `config:"path"`
	Fields    common.MapStr `config:"fields"`
	Namespace string        `config:"namespace"`
}

func defaultHttpServerConfig() HttpServerConfig {
	return HttpServerConfig{
		DefaultPath: PathConfig{
			Path:      "/",
			Namespace: "server",
		},
	}
}

func (p PathConfig) Validate() error {
	if p.Namespace == "" {
		return errors.New("`namespace` can not be empty in path configuration")
	}

	if p.Path == "" {
		return errors.New("`path` can not be empty in path configuration")
	}

	return nil
}
