package http

import (
	"errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
)

func init() {
	outputs.RegisterType("http", MakeHTTP)
}

var (
	logger = logp.NewLogger("output.http")
	// ErrNotConnected indicates failure due to client having no valid connection
	ErrNotConnected = errors.New("not connected")
	// ErrJSONEncodeFailed indicates encoding failures
	ErrJSONEncodeFailed = errors.New("json encode failed")
)

func MakeHTTP(
	_ outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return outputs.Fail(err)
	}
	tlsConfig, err := tlscommon.LoadTLSConfig(config.TLS)
	if err != nil {
		return outputs.Fail(err)
	}
	hosts, err := outputs.ReadHostList(cfg)
	if err != nil {
		return outputs.Fail(err)
	}
	proxyURL, err := parseProxyURL(config.ProxyURL)
	if err != nil {
		return outputs.Fail(err)
	}
	if proxyURL != nil {
		logger.Info("Using proxy URL: %s", proxyURL)
	}
	params := config.Params
	if len(params) == 0 {
		params = nil
	}
	clients := make([]outputs.NetworkClient, len(hosts))
	for i, host := range hosts {
		logger.Info("Making client for host: " + host)
		hostURL, err := common.MakeURL(config.Protocol, config.Path, host, 80)
		if err != nil {
			logger.Error("Invalid host param set: %s, Error: %v", host, err)
			return outputs.Fail(err)
		}
		logger.Info("Final host URL: " + hostURL)
		var client outputs.NetworkClient
		client, err = NewClient(ClientSettings{
			URL:              hostURL,
			Proxy:            proxyURL,
			TLS:              tlsConfig,
			Username:         config.Username,
			Password:         config.Password,
			Parameters:       params,
			Timeout:          config.Timeout,
			CompressionLevel: config.CompressionLevel,
			Observer:         observer,
			BatchPublish:     config.BatchPublish,
			Headers:          config.Headers,
			ContentType:      config.ContentType,
			Format:           config.Format,
		})

		if err != nil {
			return outputs.Fail(err)
		}
		client = outputs.WithBackoff(client, config.Backoff.Init, config.Backoff.Max)
		clients[i] = client
	}
	return outputs.SuccessNet(config.LoadBalance, config.BatchSize, config.MaxRetries, clients)
}
