package channel

import (
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/common"
)

// OutletFactory is used to create a new Outlet instance
type OutleterFactory func(*common.Config) (Outleter, error)

// Outleter is the outlet for a prospector
type Outleter interface {
	Close() error
	OnEvent(data *util.Data) bool
}
