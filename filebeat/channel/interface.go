package channel

import (
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/common"
)

// Factory is used to create a new Outlet instance
type Factory func(*common.Config, *common.MapStrPointer) (Outleter, error)

// Outleter is the outlet for an input
type Outleter interface {
	Close() error
	OnEvent(data *util.Data) bool
}
