// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package instrumentation

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"go.elastic.co/apm/v2"
	apmtransport "go.elastic.co/apm/v2/transport"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func init() {
	apm.DefaultTracer().Close()
}

// Instrumentation is an interface that can return an APM tracer a net.listener
type Instrumentation interface {
	Tracer() *apm.Tracer
	Listener() net.Listener
}

type instrumentation struct {
	tracer   *apm.Tracer
	listener net.Listener
}

// Tracer returns the configured tracer
// If there is not configured tracer, it returns the DefaultTracer, which is always disabled
func (t *instrumentation) Tracer() *apm.Tracer {
	if t.tracer == nil {
		return apm.DefaultTracer()
	}
	return t.tracer
}

// Listener is only relevant for APM Server sending tracing data to itself
// APM Server needs this listener to create an ad-hoc tracing server
func (t *instrumentation) Listener() net.Listener {
	return t.listener
}

// Config holds config information about self instrumenting the APM Server
type Config struct {
	Enabled     *bool           `config:"enabled"`
	Environment *string         `config:"environment"`
	Hosts       urls            `config:"hosts"`
	Profiling   ProfilingConfig `config:"profiling"`
	APIKey      string          `config:"api_key"`
	SecretToken string          `config:"secret_token"`
}

type urls []*url.URL

func (u *urls) Unpack(c interface{}) error {
	if c == nil {
		return nil
	}
	hosts, ok := c.([]interface{})
	if !ok {
		return fmt.Errorf("hosts must be a list, got: %#v", c)
	}

	nu := make(urls, len(hosts))
	for i, host := range hosts {
		h, ok := host.(string)
		if !ok {
			return fmt.Errorf("host must be a string, got: %#v", host)
		}
		url, err := url.Parse(h)
		if err != nil {
			return err
		}
		nu[i] = url
	}
	*u = nu

	return nil
}

// ProfilingConfig holds config information about self profiling the APM Server
type ProfilingConfig struct {
	CPU  *CPUProfiling  `config:"cpu"`
	Heap *HeapProfiling `config:"heap"`
}

// CPUProfiling holds config information about CPU profiling of the APM Server
type CPUProfiling struct {
	Enabled  bool          `config:"enabled"`
	Interval time.Duration `config:"interval" validate:"positive"`
	Duration time.Duration `config:"duration" validate:"positive"`
}

// IsEnabled indicates whether instrumentation is enabled
func (c *Config) IsEnabled() bool {
	return c != nil && c.Enabled != nil && *c.Enabled
}

// IsEnabled indicates whether CPU profiling is enabled
func (p *CPUProfiling) IsEnabled() bool {
	return p != nil && p.Enabled
}

// IsEnabled indicates whether heap profiling is enabled
func (p *HeapProfiling) IsEnabled() bool {
	return p != nil && p.Enabled
}

// HeapProfiling holds config information about heap profiling of the APM Server
type HeapProfiling struct {
	Enabled  bool          `config:"enabled"`
	Interval time.Duration `config:"interval" validate:"positive"`
}

// New configures and returns an instrumentation object for tracing
func New(cfg *common.Config, beatName, beatVersion string) (Instrumentation, error) {
	if !cfg.HasField("instrumentation") {
		return &instrumentation{}, nil
	}

	instrConfig, err := cfg.Child("instrumentation", -1)
	if err != nil {
		return &instrumentation{}, nil
	}

	config := Config{}

	if instrConfig == nil {
		instrConfig = common.NewConfig()
	}
	err = instrConfig.Unpack(&config)

	if err != nil {
		return nil, fmt.Errorf("could not create tracer, err: %v", err)
	}

	return initTracer(config, beatName, beatVersion)
}

func initTracer(cfg Config, beatName, beatVersion string) (*instrumentation, error) {

	logger := logp.NewLogger("tracing")

	if !cfg.IsEnabled() {
		os.Setenv("ELASTIC_APM_ACTIVE", "false")
		logger.Infof("APM instrumentation is disabled")
		return &instrumentation{}, nil
	} else {
		os.Setenv("ELASTIC_APM_ACTIVE", "true")
		logger.Infof("APM instrumentation is enabled")
	}

	if cfg.Profiling.CPU.IsEnabled() {
		interval := cfg.Profiling.CPU.Interval
		duration := cfg.Profiling.CPU.Duration
		logger.Infof("CPU profiling: every %s for %s", interval, duration)
		os.Setenv("ELASTIC_APM_CPU_PROFILE_INTERVAL", fmt.Sprintf("%dms", int(interval.Seconds()*1000)))
		os.Setenv("ELASTIC_APM_CPU_PROFILE_DURATION", fmt.Sprintf("%dms", int(duration.Seconds()*1000)))
	}
	if cfg.Profiling.Heap.IsEnabled() {
		interval := cfg.Profiling.Heap.Interval
		logger.Infof("Heap profiling: every %s", interval)
		os.Setenv("ELASTIC_APM_HEAP_PROFILE_INTERVAL", fmt.Sprintf("%dms", int(interval.Seconds()*1000)))
	}

	var tracerTransport apmtransport.Transport
	var tracerListener net.Listener

	if cfg.Hosts == nil {
		pipeListener := transport.NewPipeListener()
		pipeTransport, err := apmtransport.NewHTTPTransport(apmtransport.HTTPTransportOptions{
			ServerURLs: []*url.URL{{Scheme: "http", Host: "localhost:8200"}},
		})
		if err != nil {
			return nil, err
		}
		pipeTransport.Client.Transport = &http.Transport{
			DialContext:     pipeListener.DialContext,
			MaxIdleConns:    100,
			IdleConnTimeout: 90 * time.Second,
		}
		tracerTransport = pipeTransport
		// the traceListener will allow APM Server to create an ad-hoc server for tracing
		tracerListener = pipeListener
	} else {
		t, err := apmtransport.NewHTTPTransport(apmtransport.HTTPTransportOptions{
			APIKey:      cfg.APIKey,
			SecretToken: cfg.SecretToken,
			ServerURLs:  cfg.Hosts,
		})
		if err != nil {
			return nil, err
		}
		tracerTransport = t
	}

	var environment string
	if cfg.Environment != nil {
		environment = *cfg.Environment
	}
	tracer, err := apm.NewTracerOptions(apm.TracerOptions{
		ServiceName:        beatName,
		ServiceVersion:     beatVersion,
		ServiceEnvironment: environment,
		Transport:          tracerTransport,
	})
	if err != nil {
		return nil, err
	}

	tracer.SetLogger(warningLogger{logger})
	return &instrumentation{
		tracer:   tracer,
		listener: tracerListener,
	}, nil
}

// warningLogger wraps logp.Logger to allow to be set in the apm.Tracer.
type warningLogger struct {
	*logp.Logger
}

// Warningf logs a message at warning level.
func (l warningLogger) Warningf(format string, args ...interface{}) {
	l.Warnf(format, args...)
}
