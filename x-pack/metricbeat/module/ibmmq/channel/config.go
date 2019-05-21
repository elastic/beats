// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package channel

import (
	"github.com/felix-lessoer/beats/x-pack/metricbeat/module/ibmmq/lib"
)

var (
	DefaultConfig = ibmmqlib.Config{
		PubSub:             false,
		QMgrStat:           true,
		RemoteQueueManager: []string{""},
		Queue:        			"*",
		QueueStatus:        true,
		QueueStats:         true,
		Channel:            "*",
		Custom:             "",
		ConnectionConfig:   ibmmqlib.ConnectionConfig{
			ClientMode: false,
			UserId:     "",
			Password:   "",
		},
	}
)
