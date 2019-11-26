package syslog

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

func init() {
	outputs.RegisterType("syslog", makeSyslog)
}

func makeSyslog(
	_ outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {
	println("test .... 1")
	config := defaultConfig

	if err := cfg.Unpack(&config); err != nil {
		return outputs.Fail(err)
	}

	hosts, err := outputs.ReadHostList(cfg)
	if err != nil {
		return outputs.Fail(err)
	}

	tls, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return outputs.Fail(err)
	}

	transp := &transport.Config{
		Timeout: config.Timeout,
		Proxy:   &config.Proxy,
		TLS:     tls,
		Stats:   observer,
	}

	clients := make([]outputs.NetworkClient, len(hosts))

	for i, host := range hosts {
		var client outputs.NetworkClient

		conn, err := transport.NewClient(transp, config.Network, host, config.Port)
		if err != nil {
			return outputs.Fail(err)
		}

		client = newClient(conn, observer, config.SyslogProgram, config.SyslogPriority, config.SyslogSeverity, config.Timeout)

		client = outputs.WithBackoff(client, config.Backoff.Init, config.Backoff.Max)

		clients[i] = client
	}
	return outputs.SuccessNet(false, -1, config.MaxRetries, clients)
}
