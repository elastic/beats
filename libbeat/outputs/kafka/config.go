package kafka

import (
	"time"

	"github.com/elastic/beats/libbeat/outputs"
)

type kafkaConfig struct {
	Hosts           []string           `config:"hosts"`
	TLS             *outputs.TLSConfig `config:"tls"`
	Timeout         time.Duration      `config:"timeout"`
	Worker          int                `config:"worker"`
	UseType         bool               `config:"use_type"`
	Topic           string             `config:"topic"`
	KeepAlive       time.Duration      `config:"keep_alive"`
	MaxMessageBytes *int               `config:"max_message_bytes"`
	RequiredACKs    *int               `config:"required_acks"`
	BrokerTimeout   time.Duration      `config:"broker_timeout"`
	Compression     string             `config:"compression"`
	MaxRetries      int                `config:"max_retries"`
	ClientID        string             `config:"client_id"`
	ChanBufferSize  int                `config:"channel_buffer_size"`
}

var (
	defaultConfig = kafkaConfig{
		Hosts:           nil,
		TLS:             nil,
		Timeout:         30 * time.Second,
		Worker:          1,
		UseType:         false,
		Topic:           "",
		KeepAlive:       0,
		MaxMessageBytes: nil, // use library default
		RequiredACKs:    nil, // use library default
		BrokerTimeout:   10 * time.Second,
		Compression:     "gzip",
		MaxRetries:      3,
		ClientID:        "beats",
		ChanBufferSize:  256,
	}
)
