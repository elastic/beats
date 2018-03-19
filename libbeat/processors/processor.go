package processors

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type Processors struct {
	List []Processor
}

type Processor interface {
	Run(event *beat.Event) (*beat.Event, error)
	String() string
}

func New(config PluginConfig) (*Processors, error) {
	procs := Processors{}

	for _, processor := range config {

		if len(processor) != 1 {
			return nil, fmt.Errorf("each processor needs to have exactly one action, but found %d actions",
				len(processor))
		}

		for processorName, cfg := range processor {

			gen, exists := registry.reg[processorName]
			if !exists {
				return nil, fmt.Errorf("the processor %s doesn't exist", processorName)
			}

			cfg.PrintDebugf("Configure processor '%v' with:", processorName)
			constructor := gen.Plugin()
			plugin, err := constructor(cfg)
			if err != nil {
				return nil, err
			}

			procs.add(plugin)
		}
	}

	logp.Debug("processors", "Processors: %v", procs)
	return &procs, nil
}

func (procs *Processors) add(p Processor) {
	procs.List = append(procs.List, p)
}

// RunBC (run backwards-compatible) applies the processors, by providing the
// old interface based on common.MapStr.
// The event us temporarily converted to beat.Event. By this 'conversion' the
// '@timestamp' field can not be accessed by processors.
// Note: this method will be removed, when the publisher pipeline BC-API is to
//       be removed.
func (procs *Processors) RunBC(event common.MapStr) common.MapStr {
	ret := procs.Run(&beat.Event{Fields: event})
	if ret == nil {
		return nil
	}
	return ret.Fields
}

func (procs *Processors) All() []beat.Processor {
	if procs == nil || len(procs.List) == 0 {
		return nil
	}

	ret := make([]beat.Processor, len(procs.List))
	for i, p := range procs.List {
		ret[i] = p
	}
	return ret
}

// Applies a sequence of processing rules and returns the filtered event
func (procs *Processors) Run(event *beat.Event) *beat.Event {
	// Check if processors are set, just return event if not
	if len(procs.List) == 0 {
		return event
	}

	for _, p := range procs.List {
		var err error
		event, err = p.Run(event)
		if err != nil {
			// XXX: We don't drop the event, but continue filtering here iff the most
			//      recent processor did return an event.
			//      We want processors having this kind of implicit behavior
			//      on errors?

			logp.Debug("filter", "fail to apply processor %s: %s", p, err)
		}

		if event == nil {
			// drop event
			return nil
		}
	}

	return event
}

func (procs Processors) String() string {
	var s []string
	for _, p := range procs.List {
		s = append(s, p.String())
	}
	return strings.Join(s, ", ")
}
