package monitorcfg

import (
	"fmt"
	"github.com/elastic/beats/v7/libbeat/common"
)

type AgentInput struct {
	Id string `config:"id"`
	Name string `config:"name"`
	Meta *common.Config `config:"meta"`
	Streams []*common.Config `config:"streams" validate:"required"`
}

func (ai AgentInput) ToStandardConfig() (*common.Config, error) {
	// We expect there to be exactly one stream here, and for that config,
	// to map to a single 'regular' config.
	// to
	if len(ai.Streams) != 1 {
		return nil, fmt.Errorf("received agent config with len(streams)==%d", len(ai.Streams))
	}
	config := ai.Streams[0]

	// We overwrite the ID of monitor with the input ID since this comes
	// centrally from Kibana and should have greater precedence due to it
	// being part of a persistent store in ES that better tracks the life
	// of a config object than a text file
	if ai.Id != "" {
		err := config.SetString("id", 0, ai.Id)
		if err != nil {
			return nil, fmt.Errorf("could not override stream ID with agent ID: %w", err)
		}
	}

	return config, nil
}
