package system

import (
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper"
)

func initModule() {
	if err := helper.CheckAndEnableSeDebugPrivilege(); err != nil {
		logp.Warn("%v", err)
	}
}
