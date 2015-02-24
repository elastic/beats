package outputs

import (
	"packetbeat/common"
	"time"
)

type OutputInterface interface {
	PublishIPs(name string, localAddrs []string) error
	GetNameByIP(ip string) string
	PublishEvent(ts time.Time, event common.MapStr) error
}
