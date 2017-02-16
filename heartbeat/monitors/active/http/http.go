package http

import (
	"bytes"
	"net/http"
	"net/url"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport"

	"github.com/elastic/beats/heartbeat/monitors"
)

func init() {
	monitors.RegisterActive("http", create)
}

var debugf = logp.MakeDebug("http")

func create(
	info monitors.Info,
	cfg *common.Config,
) ([]monitors.Job, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	tls, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, err
	}

	var body []byte
	var enc contentEncoder

	if config.Check.Request.SendBody != "" {
		var err error
		compression := config.Check.Request.Compression
		enc, err = getContentEncoder(compression.Type, compression.Level)
		if err != nil {
			return nil, err
		}

		buf := bytes.NewBuffer(nil)
		err = enc.Encode(buf, bytes.NewBufferString(config.Check.Request.SendBody))
		if err != nil {
			return nil, err
		}

		body = buf.Bytes()
	}

	validator := makeValidateResponse(&config.Check.Response)

	jobs := make([]monitors.Job, len(config.URLs))

	if config.ProxyURL != "" {
		transport, err := newRoundTripper(&config, tls)
		if err != nil {
			return nil, err
		}

		for i, url := range config.URLs {
			jobs[i], err = newHTTPMonitorHostJob(url, &config, transport, enc, body, validator)
			if err != nil {
				return nil, err
			}
		}
	} else {
		for i, url := range config.URLs {
			jobs[i], err = newHTTPMonitorIPsJob(&config, url, tls, enc, body, validator)
			if err != nil {
				return nil, err
			}
		}
	}

	return jobs, nil
}

func newRoundTripper(config *Config, tls *transport.TLSConfig) (*http.Transport, error) {
	var proxy func(*http.Request) (*url.URL, error)
	if config.ProxyURL != "" {
		url, err := url.Parse(config.ProxyURL)
		if err != nil {
			return nil, err
		}
		proxy = http.ProxyURL(url)
	}

	dialer := transport.NetDialer(config.Timeout)
	tlsDialer, err := transport.TLSDialer(dialer, tls, config.Timeout)
	if err != nil {
		return nil, err
	}

	return &http.Transport{
		Proxy:             proxy,
		Dial:              dialer.Dial,
		DialTLS:           tlsDialer.Dial,
		DisableKeepAlives: true,
	}, nil
}
