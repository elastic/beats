package kafka

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/outputs"
)

type kafkaConfig struct {
	Hosts           []string           `config:"hosts"               validate:"required"`
	TLS             *outputs.TLSConfig `config:"tls"`
	Timeout         time.Duration      `config:"timeout"             validate:"min=1"`
	Worker          int                `config:"worker"              validate:"min=1"`
	UseType         bool               `config:"use_type"`
	Topic           string             `config:"topic"`
	KeepAlive       time.Duration      `config:"keep_alive"          validate:"min=0"`
	MaxMessageBytes *int               `config:"max_message_bytes"   validate:"min=1"`
	RequiredACKs    *int               `config:"required_acks"       validate:"min=-1"`
	BrokerTimeout   time.Duration      `config:"broker_timeout"      validate:"min=1"`
	Compression     string             `config:"compression"`
	MaxRetries      int                `config:"max_retries"`
	ClientID        string             `config:"client_id"`
	ChanBufferSize  int                `config:"channel_buffer_size" validate:"min=1"`
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

func (c *kafkaConfig) Validate() error {
	if len(c.Hosts) == 0 {
		return errors.New("no hosts configured")
	}

	if c.UseType == false && c.Topic == "" {
		return errors.New("use_type must be true or topic must be set")
	}

	if _, ok := compressionModes[strings.ToLower(c.Compression)]; !ok {
		return fmt.Errorf("compression mode '%v' unknown", c.Compression)
	}

	return nil
}
