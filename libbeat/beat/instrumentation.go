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

package beat

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"go.elastic.co/apm"
	"go.elastic.co/apm/transport"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/pipe"
)

func init() {
	// we need to close the default tracer to prevent the beat sending events to localhost:8200
	apm.DefaultTracer.Close()
}

// Instrumentation holds an APM tracer a net.Listener
type Instrumentation struct {
	tracer *apm.Tracer
	// Listener is only relevant for APM Server sending tracing data to itself
	// APM Server needs this Listener to create an ad-hoc tracing server
	Listener net.Listener
}

// GetTracer returns the configured tracer
// If there is not configured tracer, it returns the DefaultTracer, which is always disabled
func (t *Instrumentation) GetTracer() *apm.Tracer {
	if t == nil || t.tracer == nil {
		return apm.DefaultTracer
	}
	return t.tracer
}

// InstrumentationConfig holds config information about self instrumenting the APM Server
type InstrumentationConfig struct {
	Enabled     *bool           `config:"enabled"`
	Environment *string         `config:"environment"`
	Hosts       []*url.URL      `config:"hosts"`
	Profiling   ProfilingConfig `config:"profiling"`
	APIKey      string          `config:"api_key"`
	SecretToken string          `config:"secret_token"`
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
func (c *InstrumentationConfig) IsEnabled() bool {
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

// CreateInstrumentation configures and returns an instrumentation object for tracing
func CreateInstrumentation(cfg *common.Config, info Info) (*Instrumentation, error) {
	if !cfg.HasField("instrumentation") {
		return nil, nil
	}

	instrumentation, err := cfg.Child("instrumentation", -1)
	if err != nil {
		return nil, nil
	}

	config := InstrumentationConfig{}

	if instrumentation == nil {
		instrumentation = common.NewConfig()
	}
	err = instrumentation.Unpack(&config)

	if err != nil {
		return nil, fmt.Errorf("could not create tracer, err: %v", err)
	}

	return initTracer(config, info)
}

func initTracer(cfg InstrumentationConfig, info Info) (*Instrumentation, error) {

	logger := logp.NewLogger("tracing")

	if !cfg.IsEnabled() {
		os.Setenv("ELASTIC_APM_ACTIVE", "false")
		logger.Infof("APM instrumentation is disabled")
		return nil, nil
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

	var tracerTransport transport.Transport
	var tracerListener net.Listener
	if cfg.Hosts != nil {
		// tracing destined for host explicitly set
		t, err := transport.NewHTTPTransport()

		if err != nil {
			return nil, err
		}
		t.SetServerURL(cfg.Hosts...)
		if cfg.APIKey != "" {
			t.SetAPIKey(cfg.APIKey)
		} else {
			t.SetSecretToken(cfg.SecretToken)
		}
		tracerTransport = t
		logger.Infof("APM tracer directed to %s", cfg.Hosts)
	} else {
		// if the host is not set, the running beat is assumed to be an APM Server sending traces to itself
		// if another beat is running without an APM Server host configured, we default to localhost
		pipeListener := pipe.NewListener()
		pipeTransport, err := transport.NewHTTPTransport()
		if err != nil {
			return nil, err
		}
		pipeTransport.SetServerURL(&url.URL{Scheme: "http", Host: "localhost"})
		pipeTransport.Client.Transport = &http.Transport{
			DialContext:     pipeListener.DialContext,
			MaxIdleConns:    100,
			IdleConnTimeout: 90 * time.Second,
		}
		tracerTransport = pipeTransport
		// the traceListener will allow APM Server to create a ad-hoc server for tracing
		tracerListener = pipeListener
	}

	var environment string
	if cfg.Environment != nil {
		environment = *cfg.Environment
	}
	tracer, err := apm.NewTracerOptions(apm.TracerOptions{
		ServiceName:        info.Beat,
		ServiceVersion:     info.Version,
		ServiceEnvironment: environment,
		Transport:          tracerTransport,
	})
	if tracer != nil {
		tracer.SetLogger(logger)
	}

	return &Instrumentation{
		tracer:   tracer,
		Listener: tracerListener,
	}, err
}
