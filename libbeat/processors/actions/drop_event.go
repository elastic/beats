package actions

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

type dropEvent struct{}

func init() {
	processors.RegisterPlugin("drop_event",
		configChecked(newDropEvent, allowedFields("when")))
}

var dropEventsSingleton = (*dropEvent)(nil)

func newDropEvent(c *common.Config) (processors.Processor, error) {
	return dropEventsSingleton, nil
}

func (*dropEvent) Run(_ *beat.Event) (*beat.Event, error) {
	// return event=nil to delete the entire event
	return nil, nil
}

func (*dropEvent) String() string { return "drop_event" }
