package common

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/common/cfgtype"
)

// ListenerConfig exposes the shared listener configuration.
type ListenerConfig struct {
	Timeout        time.Duration
	MaxMessageSize cfgtype.ByteSize
	MaxConnections int
}
