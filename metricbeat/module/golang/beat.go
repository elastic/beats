// Beat module and metric
package golang

import (
	"github.com/elastic/beats/metricbeat/helper"
)

// This one comabines module and metric
func init() {
	helper.Registry.AddModuler("golang", Golang{})
	//helper.NewModule("golang", Golang{}).Register()
}

type Golang struct {
}

func (b Golang) Setup() error {
	return nil
}
