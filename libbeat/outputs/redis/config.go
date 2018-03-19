package redis

import (
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

type redisConfig struct {
	Password    string                `config:"password"`
	Index       string                `config:"index"`
	Key         string                `config:"key"`
	Port        int                   `config:"port"`
	LoadBalance bool                  `config:"loadbalance"`
	Timeout     time.Duration         `config:"timeout"`
	BulkMaxSize int                   `config:"bulk_max_size"`
	MaxRetries  int                   `config:"max_retries"`
	TLS         *outputs.TLSConfig    `config:"ssl"`
	Proxy       transport.ProxyConfig `config:",inline"`
	Codec       codec.Config          `config:"codec"`
	Db          int                   `config:"db"`
	DataType    string                `config:"datatype"`
}

var (
	defaultConfig = redisConfig{
		Port:        6379,
		LoadBalance: true,
		Timeout:     5 * time.Second,
		BulkMaxSize: 2048,
		MaxRetries:  3,
		TLS:         nil,
		Db:          0,
		DataType:    "list",
	}
)

func (c *redisConfig) Validate() error {
	switch c.DataType {
	case "", "list", "channel":
	default:
		return fmt.Errorf("redis data type %v not supported", c.DataType)
	}

	if c.Key != "" && c.Index != "" {
		return errors.New("Cannot use both `output.redis.key` and `output.redis.index` configuration options." +
			" Set only `output.redis.key`")
	}

	if c.Key == "" && c.Index != "" {
		c.Key = c.Index
		logp.Warn("The `output.redis.index` configuration setting is deprecated. Use `output.redis.key` instead.")
	}

	return nil
}
