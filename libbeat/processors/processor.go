package processors

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type Processors struct {
	list []Processor
}

func New(config PluginConfig) (*Processors, error) {

	processors := Processors{}

	for _, processor := range config {

		if len(processor) != 1 {
			return nil, fmt.Errorf("each processor needs to have exactly one action, but found %d actions.",
				len(processor))
		}

		for processorName, cfg := range processor {

			constructor, exists := constructors[processorName]
			if !exists {
				return nil, fmt.Errorf("the processor %s doesn't exist", processorName)
			}

			plugin, err := constructor(cfg)
			if err != nil {
				return nil, err
			}

			processors.addProcessor(plugin)
		}
	}

	logp.Debug("processors", "Processors: %v", processors)
	return &processors, nil
}

func (processors *Processors) addProcessor(p Processor) {

	processors.list = append(processors.list, p)
}

// Applies a sequence of processing rules and returns the filtered event
func (processors *Processors) Run(event common.MapStr) common.MapStr {

	// Check if processors are set, just return event if not
	if len(processors.list) == 0 {
		return event
	}

	// clone the event at first, before starting filtering
	filtered := event.Clone()
	var err error

	for _, p := range processors.list {
		filtered, err = p.Run(filtered)
		if err != nil {
			logp.Debug("filter", "fail to apply processor %s: %s", p, err)
		}
		if filtered == nil {
			// drop event
			return nil
		}
	}

	return filtered
}

func (processors Processors) String() string {
	s := []string{}

	for _, p := range processors.list {

		s = append(s, p.String())
	}
	return strings.Join(s, ", ")
}
