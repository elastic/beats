package actions

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

type dropEvent struct{}

func init() {
	processors.RegisterPlugin("drop_event",
		configChecked(newDropEvent, allowedFields("when")))
}

func newDropEvent(c common.Config) (processors.Processor, error) {
	return dropEvent{}, nil
}

func (f dropEvent) Run(event common.MapStr) (common.MapStr, error) {
	// return event=nil to delete the entire event
	return nil, nil
}

func (f dropEvent) String() string { return "drop_event" }
