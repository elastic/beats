package socket

import "time"

// Config is the configuration specific to the socket MetricSet.
type Config struct {
	ReverseLookup *ReverseLookupConfig `config:"socket.reverse_lookup"`
}

// ReverseLookupConfig contains the configuration that controls the reverse
// DNS lookup behavior.
type ReverseLookupConfig struct {
	Enabled    *bool         `config:"enabled"`
	SuccessTTL time.Duration `config:"success_ttl"`
	FailureTTL time.Duration `config:"failure_ttl"`
}

// IsEnabled returns true if reverse_lookup is defined and 'enabled' is either
// not set or set to true.
func (c *ReverseLookupConfig) IsEnabled() bool {
	return c != nil && (c.Enabled == nil || *c.Enabled)
}

const (
	defSuccessTTL = 60 * time.Second
	defFailureTTL = 60 * time.Second
)

var defaultConfig = Config{
	ReverseLookup: nil,
}
