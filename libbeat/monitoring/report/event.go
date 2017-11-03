package report

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
)

type Event struct {
	Timestamp time.Time     `struct:"timestamp"`
	Fields    common.MapStr `struct:",inline"`
}
