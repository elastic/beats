package redis

import (
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

type redisOut struct {
	mode mode.ConnectionMode
	topology
}

var (
	debugf = logp.MakeDebug("redis")
)

const (
	defaultWaitRetry    = 1 * time.Second
	defaultMaxWaitRetry = 60 * time.Second
)

func init() {
	outputs.RegisterOutputPlugin("redis", new)
}

func new(cfg *common.Config, expireTopo int) (outputs.Outputer, error) {
	r := &redisOut{}
	if err := r.init(cfg, expireTopo); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *redisOut) init(cfg *common.Config, expireTopo int) error {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return err
	}

	sendRetries := config.MaxRetries
	maxAttempts := config.MaxRetries + 1
	if sendRetries < 0 {
		maxAttempts = 0
	}

	var dataType redisDataType
	switch config.DataType {
	case "", "list":
		dataType = redisListType
	case "channel":
		dataType = redisChannelType
	default:
		return errors.New("Bad Redis data type")
	}

	index := []byte(config.Index)
	if len(index) == 0 {
		return fmt.Errorf("missing %v", cfg.PathOf("index"))
	}

	tls, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return err
	}

	transp := &transport.Config{
		Timeout: config.Timeout,
		Proxy:   &config.Proxy,
		TLS:     tls,
	}

	// configure topology support
	r.topology.init(transp, topoConfig{
		host:     config.HostTopology,
		password: config.PasswordTopology,
		db:       config.DbTopology,
		expire:   time.Duration(expireTopo) * time.Second,
	})

	// configure publisher clients
	clients, err := mode.MakeClients(cfg, func(host string) (mode.ProtocolClient, error) {
		t, err := transport.NewClient(transp, "tcp", host, config.Port)
		if err != nil {
			return nil, err
		}
		return newClient(t, config.Password, config.Db, index, dataType), nil
	})
	if err != nil {
		return err
	}

	logp.Info("Max Retries set to: %v", sendRetries)
	m, err := mode.NewConnectionMode(clients, !config.LoadBalance,
		maxAttempts, defaultWaitRetry, config.Timeout, defaultMaxWaitRetry)
	if err != nil {
		return err
	}

	r.mode = m
	return nil
}

func (r *redisOut) Close() error {
	return r.mode.Close()
}

func (r *redisOut) PublishEvent(
	signaler op.Signaler,
	opts outputs.Options,
	event common.MapStr,
) error {
	return r.mode.PublishEvent(signaler, opts, event)
}

func (r *redisOut) BulkPublish(
	signaler op.Signaler,
	opts outputs.Options,
	events []common.MapStr,
) error {
	return r.mode.PublishEvents(signaler, opts, events)
}
