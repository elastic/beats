package logstash

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

const (
	minWindowSize             int = 1
	defaultStartMaxWindowSize int = 10
)

var (
	logstashMetrics = outputs.Metrics.NewRegistry("logstash")

	ackedEvents            = monitoring.NewInt(logstashMetrics, "events.acked")
	eventsNotAcked         = monitoring.NewInt(logstashMetrics, "events.not_acked")
	publishEventsCallCount = monitoring.NewInt(logstashMetrics, "publishEvents.call.count")

	statReadBytes   = monitoring.NewInt(logstashMetrics, "read.bytes")
	statWriteBytes  = monitoring.NewInt(logstashMetrics, "write.bytes")
	statReadErrors  = monitoring.NewInt(logstashMetrics, "read.errors")
	statWriteErrors = monitoring.NewInt(logstashMetrics, "write.errors")
)

var debugf = logp.MakeDebug("logstash")

func init() {
	outputs.RegisterType("logstash", makeLogstash)
}

func makeLogstash(beat common.BeatInfo, cfg *common.Config) (outputs.Group, error) {
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
		Stats: &transport.IOStats{
			Read:               statReadBytes,
			Write:              statWriteBytes,
			ReadErrors:         statReadErrors,
			WriteErrors:        statWriteErrors,
			OutputsWrite:       outputs.WriteBytes,
			OutputsWriteErrors: outputs.WriteErrors,
		},
	}

	clients := make([]outputs.NetworkClient, len(hosts))
	for i, host := range hosts {
		var client outputs.NetworkClient

		conn, err := transport.NewClient(transp, "tcp", host, config.Port)
		if err != nil {
			return outputs.Fail(err)
		}

		if config.Pipelining > 0 {
			client, err = newAsyncClient(conn, config)
		} else {
			client, err = newSyncClient(conn, config)
		}
		if err != nil {
			return outputs.Fail(err)
		}

		client = outputs.WithBackoff(client, config.Backoff.Init, config.Backoff.Max)
		clients[i] = client
	}

	return outputs.SuccessNet(config.LoadBalance, config.BulkMaxSize, config.MaxRetries, clients)
}
