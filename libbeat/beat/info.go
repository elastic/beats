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
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid/v5"
	"go.opentelemetry.io/collector/consumer"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// Info stores a beats instance meta data.
type Info struct {
	Beat            string    // The actual beat's name
	IndexPrefix     string    // The beat's index prefix in Elasticsearch.
	Version         string    // The beat version. Defaults to the libbeat version when an implementation does not set a version
	ElasticLicensed bool      // Whether the beat is licensed under and Elastic License
	Name            string    // configured beat name
	Hostname        string    // hostname
	FQDN            string    // FQDN
	ID              uuid.UUID // ID assigned to beat machine
	EphemeralID     uuid.UUID // ID assigned to beat process invocation (PID)
	FirstStart      time.Time // The time of the first start of the Beat.
	StartTime       time.Time // The time of last start of the Beat. Updated when the Beat is started or restarted.
	UserAgent       string    // A string of the user-agent that can be passed to any outputs or network connections

	// Monitoring-related fields
	Monitoring           Monitoring
	LogConsumer          consumer.Logs // otel log consumer
	UseDefaultProcessors bool          // Whether to use the default processors
	Logger               *logp.Logger
}

type Monitoring struct {
	DefaultUsername string // The default username to be used to connect to Elasticsearch Monitoring

	Namespace     *monitoring.Namespace // a monitor namespace that is unique per beat instance
	InfoRegistry  *monitoring.Registry
	StateRegistry *monitoring.Registry
	StatsRegistry *monitoring.Registry

	// InputHTTPMetrics manages the input metrics to be published by the HTTP
	// monitoring endpoint. It replaces the deprecated inputmon.NewInputRegistry.
	InputHTTPMetrics *InputHTTPMetrics
}

func (i Info) FQDNAwareHostname(useFQDN bool) string {
	if useFQDN {
		return i.FQDN
	}

	return i.Hostname
}

// NamespaceRegistry returns the monitoring registry from Namespace.
// If Namespace isn't set, it returns a new registry associated to no namespace
// for every call.
func (m *Monitoring) NamespaceRegistry() *monitoring.Registry {
	if m.Namespace == nil {
		return monitoring.NewRegistry()
	}

	return m.Namespace.GetRegistry()
}

// SetupRegistries sets up the monitoring registries.
// If Namespace is nil, a namespace is created for each registry.
// If Namespace is non-nil, then the registries are created on Namespace.
func (m *Monitoring) SetupRegistries() {
	var infoRegistry *monitoring.Registry
	var stateRegistry *monitoring.Registry
	var statsRegistry *monitoring.Registry

	if m.Namespace != nil {
		reg := m.Namespace.GetRegistry()
		infoRegistry = reg.GetRegistry("info")
		if infoRegistry == nil {
			infoRegistry = reg.NewRegistry("info")
		}

		stateRegistry = reg.GetRegistry("state")
		if stateRegistry == nil {
			stateRegistry = reg.NewRegistry("state")
		}

		statsRegistry = reg.GetRegistry("stats")
		if statsRegistry == nil {
			statsRegistry = reg.NewRegistry("stats")
		}
	} else {
		infoRegistry = monitoring.GetNamespace("info").GetRegistry()
		stateRegistry = monitoring.GetNamespace("state").GetRegistry()
		statsRegistry = monitoring.GetNamespace("stats").GetRegistry()
	}

	m.InfoRegistry = infoRegistry
	m.StateRegistry = stateRegistry
	m.StatsRegistry = statsRegistry
}

// InputHTTPMetrics manages the inputs metrics registries to be published by the
// HTTP monitoring endpoint.
type InputHTTPMetrics struct {
	mu         sync.RWMutex
	registries map[string]*monitoring.Registry
}

// NewInputHTTPMetrics returns a new InputHTTPMetrics
func NewInputHTTPMetrics() *InputHTTPMetrics {
	return &InputHTTPMetrics{
		registries: map[string]*monitoring.Registry{},
	}
}

// RegisterMetrics adds reg to the collection of registries. The registry must
// have at least a `id` and `input` string variables, otherwise the registry is
// rejected and an error is returned.
//
// The registry is associated with the value of its `id` string variable,
// calling UnregisterMetrics(id) will unregister reg.
//
// If an id registry has been already registered, it'll be overridden.
// When the input finishes, it should call UnregisterMetrics to release the
// associated resources.
func (i *InputHTTPMetrics) RegisterMetrics(reg *monitoring.Registry) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	id := getStringVar(reg.Get("id"))
	input := getStringVar(reg.Get("input"))

	var errMgs []string
	if id == "" {
		errMgs = append(errMgs, "'id' empty or absent")
	}
	if input == "" {
		errMgs = append(errMgs, "'input' empty or absent")
	}
	if len(errMgs) > 0 {
		return errors.New("invalid metrics registry: " +
			strings.Join(errMgs, ", "))
	}

	i.registries[id] = reg

	return nil
}

// UnregisterMetrics removes the registry identified by id.
func (i *InputHTTPMetrics) UnregisterMetrics(id string) {
	i.mu.Lock()
	defer i.mu.Unlock()

	delete(i.registries, id)
}

// CollectStructSnapshot returns the result of calling
// monitoring.CollectStructSnapshot with mode Full and expvar false on each of
// the registered metrics registry. It returns a map using the metrics registry
// id as key and the structured snapshot is the associated value.
func (i *InputHTTPMetrics) CollectStructSnapshot() map[string]map[string]any {
	i.mu.Lock()
	defer i.mu.Unlock()

	registeredInputRegistries := map[string]map[string]any{}
	for id, reg := range i.registries {
		registeredInputRegistries[id] = monitoring.CollectStructSnapshot(
			reg, monitoring.Full, false)
	}

	return registeredInputRegistries
}

func getStringVar(v monitoring.Var) string {
	if v != nil {
		if s, ok := v.(*monitoring.String); ok {
			return s.Get()
		}
	}

	return ""
}
