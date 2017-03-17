package elasticsearch

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/monitoring"
)

func makeSnapshot(R *monitoring.Registry) common.MapStr {
	mode := monitoring.Full
	return common.MapStr(monitoring.CollectStructSnapshot(R, mode, false))
}
