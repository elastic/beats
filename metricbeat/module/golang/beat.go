// Beat module and metric
package golang

import (
	"github.com/elastic/beats/metricbeat/helper"
)

// This one comabines module and metric
func init() {
	Module.Register()
}

// Module object
var Module = helper.NewModule("golang", Golang{})

type Golang struct {
}

func (b Golang) Setup() error {
	return nil
}
