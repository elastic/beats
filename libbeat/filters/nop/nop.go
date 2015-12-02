// Package nop implements a Packetbeat filter that does
// absolutely nothing.
package nop

import (
	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/filters"
)

type Nop struct {
	name string
}

func (nop *Nop) New(name string, config map[string]interface{}) (filters.FilterPlugin, error) {
	return &Nop{name: name}, nil
}

func (nop *Nop) Filter(event common.MapStr) (common.MapStr, error) {
	return event, nil
}

func (nop *Nop) String() string {
	return nop.name
}

func (nop *Nop) Type() filters.Filter {
	return filters.NopFilter
}
