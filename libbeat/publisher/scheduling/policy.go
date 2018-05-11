package scheduling

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

type PolicyFactory func(cfg *common.Config) (Policy, error)

type Policy interface {
	Connect(ctx Context) (Handler, error)
}

type Handler interface {
	OnEvent(beat.Event) (beat.Event, error)
}
