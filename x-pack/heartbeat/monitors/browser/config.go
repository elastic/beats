package browser

import (
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser/source"
)

type Config struct {
	Schedule string                 `config:"schedule"`
	Params   map[string]interface{} `config:"params"`
	RawConfig *common.Config
	Source *source.Source `config:"source"`
}



