package monitorcfg

import (
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type AgentInput struct {
	Id string `config:"id"`
	Name string `config:"name"`
	Meta AgentMeta `config:"meta"`
	Streams []*common.Config `config:"streams" validate:"required"`
}

type AgentMeta struct {
	Pkg AgentPackage `config:"package"`
}

type AgentPackage struct {
	Name string `config:"name"`
	Version string `config:"version"`
}

func normalizeConfig(config *common.Config) *common.Config {
	logp.Warn("NORMALIZE CONFIG %s", config)
	return config
}

