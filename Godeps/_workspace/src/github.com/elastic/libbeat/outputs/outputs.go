package outputs

import (
	"time"

	"github.com/elastic/libbeat/common"
)

type OutputInterface interface {
	PublishIPs(name string, localAddrs []string) error
	GetNameByIP(ip string) string
	PublishEvent(ts time.Time, event common.MapStr) error
}
