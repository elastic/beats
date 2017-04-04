package elasticsearch

import (
	"time"

	"github.com/elastic/beats/libbeat/outputs"
)

// config is subset of libbeat/outputs/elasticsearch config tailored
// for reporting metrics only
type config struct {
	Hosts            []string
	Protocol         string
	Params           map[string]string  `config:"parameters"`
	Headers          map[string]string  `config:"headers"`
	Username         string             `config:"username"`
	Password         string             `config:"password"`
	ProxyURL         string             `config:"proxy_url"`
	CompressionLevel int                `config:"compression_level" validate:"min=0, max=9"`
	TLS              *outputs.TLSConfig `config:"ssl"`
	MaxRetries       int                `config:"max_retries"`
	Timeout          time.Duration      `config:"timeout"`
	Period           time.Duration      `config:"period"`
	BulkMaxSize      int                `config:"bulk_max_size" validate:"min=0"`
	BufferSize       int                `config:"buffer_size"`
	Tags             []string           `config:"tags"`
}

var defaultConfig = config{
	Hosts:            nil,
	Protocol:         "http",
	Params:           nil,
	Headers:          nil,
	Username:         "beats_system",
	Password:         "",
	ProxyURL:         "",
	CompressionLevel: 0,
	TLS:              nil,
	MaxRetries:       3,
	Timeout:          60 * time.Second,
	Period:           10 * time.Second,
	BulkMaxSize:      50,
	BufferSize:       50,
	Tags:             nil,
}
