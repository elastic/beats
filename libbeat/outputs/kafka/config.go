package kafka

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/fmtstr"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codec"
)

type kafkaConfig struct {
	Hosts           []string                  `config:"hosts"               validate:"required"`
	TLS             *outputs.TLSConfig        `config:"ssl"`
	Timeout         time.Duration             `config:"timeout"             validate:"min=1"`
	Metadata        metaConfig                `config:"metadata"`
	Key             *fmtstr.EventFormatString `config:"key"`
	Partition       map[string]*common.Config `config:"partition"`
	KeepAlive       time.Duration             `config:"keep_alive"          validate:"min=0"`
	MaxMessageBytes *int                      `config:"max_message_bytes"   validate:"min=1"`
	RequiredACKs    *int                      `config:"required_acks"       validate:"min=-1"`
	BrokerTimeout   time.Duration             `config:"broker_timeout"      validate:"min=1"`
	Compression     string                    `config:"compression"`
	Version         string                    `config:"version"`
	BulkMaxSize     int                       `config:"bulk_max_size"`
	MaxRetries      int                       `config:"max_retries"         validate:"min=-1,nonzero"`
	ClientID        string                    `config:"client_id"`
	ChanBufferSize  int                       `config:"channel_buffer_size" validate:"min=1"`
	Username        string                    `config:"username"`
	Password        string                    `config:"password"`
	Codec           codec.Config              `config:"codec"`
}

type metaConfig struct {
	Retry       metaRetryConfig `config:"retry"`
	RefreshFreq time.Duration   `config:"refresh_frequency" validate:"min=0"`
}

type metaRetryConfig struct {
	Max     int           `config:"max"     validate:"min=0"`
	Backoff time.Duration `config:"backoff" validate:"min=0"`
}

var (
	defaultConfig = kafkaConfig{
		Hosts:       nil,
		TLS:         nil,
		Timeout:     30 * time.Second,
		BulkMaxSize: 2048,
		Metadata: metaConfig{
			Retry: metaRetryConfig{
				Max:     3,
				Backoff: 250 * time.Millisecond,
			},
			RefreshFreq: 10 * time.Minute,
		},
		KeepAlive:       0,
		MaxMessageBytes: nil, // use library default
		RequiredACKs:    nil, // use library default
		BrokerTimeout:   10 * time.Second,
		Compression:     "gzip",
		Version:         "",
		MaxRetries:      3,
		ClientID:        "beats",
		ChanBufferSize:  256,
		Username:        "",
		Password:        "",
	}
)

func (c *kafkaConfig) Validate() error {
	if len(c.Hosts) == 0 {
		return errors.New("no hosts configured")
	}

	if _, ok := compressionModes[strings.ToLower(c.Compression)]; !ok {
		return fmt.Errorf("compression mode '%v' unknown", c.Compression)
	}

	if _, ok := kafkaVersions[c.Version]; !ok {
		return fmt.Errorf("unknown/unsupported kafka version '%v'", c.Version)
	}

	if c.Username != "" && c.Password == "" {
		return fmt.Errorf("password must be set when username is configured")
	}

	return nil
}
