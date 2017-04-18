package actions

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

type addLocale struct{}

func init() {
	processors.RegisterPlugin("add_locale", newAddLocale)
}

func newAddLocale(c common.Config) (processors.Processor, error) {
	return addLocale{}, nil
}

func (l addLocale) Run(event common.MapStr) (common.MapStr, error) {
	zone, _ := time.Now().Zone()
	event.Put("beat.timezone", zone)

	return event, nil
}

func (l addLocale) String() string {
	return "add_locale"
}
