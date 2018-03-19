package logstash

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

const (
	minWindowSize             int = 1
	defaultStartMaxWindowSize int = 10
)

var debugf = logp.MakeDebug("logstash")

func init() {
	outputs.RegisterType("logstash", makeLogstash)
}

func makeLogstash(
	beat beat.Info,
	observer outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {
	if !cfg.HasField("index") {
		cfg.SetString("index", -1, beat.Beat)
	}

	config := newConfig()
	if err := cfg.Unpack(config); err != nil {
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

		conn, err := transport.NewClient(transp, "tcp", host, config.Port)
		if err != nil {
			return outputs.Fail(err)
		}

		if config.Pipelining > 0 {
			client, err = newAsyncClient(beat, conn, observer, config)
		} else {
			client, err = newSyncClient(beat, conn, observer, config)
		}
		if err != nil {
			return outputs.Fail(err)
		}

		client = outputs.WithBackoff(client, config.Backoff.Init, config.Backoff.Max)
		clients[i] = client
	}

	return outputs.SuccessNet(config.LoadBalance, config.BulkMaxSize, config.MaxRetries, clients)
}
