package syslog

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

const (
	defaultWaitRetry    = 1 * time.Second
	defaultMaxWaitRetry = 60 * time.Second
)

func init() {
	outputs.RegisterOutputPlugin("syslog", New)
}

type syslog struct {
	mode mode.ConnectionMode
}

func New(cfg *common.Config, _ int) (outputs.Outputer, error) {
	output := &syslog{}
	err := output.init(cfg)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func makeClientFactory(cfg *config, tcfg *transport.Config) mode.ClientFactory {
	return func(host string) (mode.ProtocolClient, error) {
		t, err := transport.NewClient(tcfg, "tcp", host, cfg.Port)
		if err != nil {
			return nil, err
		}
		program := cfg.SyslogProgram
		priority := cfg.SyslogPriority
		severity := cfg.SyslogSeverity
		return newClient(t, program, priority, severity), nil
	}
}

func (s *syslog) init(cfg *common.Config) error {
	config := defaultConfig
	sendRetries := config.MaxRetries
	maxAttempts := sendRetries + 1

	if err := cfg.Unpack(&config); err != nil {
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
	}

	clients, err := mode.MakeClients(cfg, makeClientFactory(&config, transp))

	m, err := mode.NewConnectionMode(clients, false, maxAttempts,
		defaultWaitRetry, config.Timeout, defaultMaxWaitRetry)
	if err != nil {
		return err
	}

	s.mode = m

	return nil
}

// Implement Outputer
func (c *syslog) Close() error {
	return c.mode.Close()
}

func (c *syslog) PublishEvent(signaler op.Signaler, opts outputs.Options, event common.MapStr) error {
	return c.mode.PublishEvent(signaler, opts, event)
}
