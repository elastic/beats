package redis

import (
	"errors"
	"expvar"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
	"github.com/elastic/beats/libbeat/outputs/mode/modeutil"
	"github.com/elastic/beats/libbeat/outputs/outil"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

type redisOut struct {
	mode mode.ConnectionMode
	topology
	beatName string
}

var debugf = logp.MakeDebug("redis")

// Metrics that can retrieved through the expvar web interface.
var (
	statReadBytes   = expvar.NewInt("libbeat.redis.publish.read_bytes")
	statWriteBytes  = expvar.NewInt("libbeat.redis.publish.write_bytes")
	statReadErrors  = expvar.NewInt("libbeat.redis.publish.read_errors")
	statWriteErrors = expvar.NewInt("libbeat.redis.publish.write_errors")
)

const (
	defaultWaitRetry    = 1 * time.Second
	defaultMaxWaitRetry = 60 * time.Second
)

func init() {
	outputs.RegisterOutputPlugin("redis", new)
}

func new(beatName string, cfg *common.Config, expireTopo int) (outputs.Outputer, error) {
	r := &redisOut{beatName: beatName}
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

	if cfg.HasField("index") && !cfg.HasField("key") {
		s, err := cfg.String("index", -1)
		if err != nil {
			return err
		}
		if err := cfg.SetString("key", -1, s); err != nil {
			return err
		}
	}
	if !cfg.HasField("key") {
		cfg.SetString("key", -1, r.beatName)
	}

	key, err := outil.BuildSelectorFromConfig(cfg, outil.Settings{
		Key:              "key",
		MultiKey:         "keys",
		EnableSingleOnly: true,
		FailEmpty:        true,
	})
	if err != nil {
		return err
	}

	tls, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return err
	}

	transp := &transport.Config{
		Timeout: config.Timeout,
		Proxy:   &config.Proxy,
		TLS:     tls,
		Stats: &transport.IOStats{
			Read:        statReadBytes,
			Write:       statWriteBytes,
			ReadErrors:  statReadErrors,
			WriteErrors: statWriteErrors,
		},
	}

	// configure topology support
	r.topology.init(transp, topoConfig{
		host:     config.HostTopology,
		password: config.PasswordTopology,
		db:       config.DbTopology,
		expire:   time.Duration(expireTopo) * time.Second,
	})

	// configure publisher clients
	clients, err := modeutil.MakeClients(cfg, func(host string) (mode.ProtocolClient, error) {
		t, err := transport.NewClient(transp, "tcp", host, config.Port)
		if err != nil {
			return nil, err
		}
		return newClient(t, config.Password, config.Db, key, dataType), nil
	})
	if err != nil {
		return err
	}

	logp.Info("Max Retries set to: %v", sendRetries)
	m, err := modeutil.NewConnectionMode(clients, modeutil.Settings{
		Failover:     !config.LoadBalance,
		MaxAttempts:  maxAttempts,
		Timeout:      config.Timeout,
		WaitRetry:    defaultWaitRetry,
		MaxWaitRetry: defaultMaxWaitRetry,
	})
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
	data outputs.Data,
) error {
	return r.mode.PublishEvent(signaler, opts, data)
}

func (r *redisOut) BulkPublish(
	signaler op.Signaler,
	opts outputs.Options,
	data []outputs.Data,
) error {
	return r.mode.PublishEvents(signaler, opts, data)
}
