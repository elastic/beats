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

package management

import (
	"sync"

	"github.com/gofrs/uuid"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
)

// Status describes the current status of the beat.
type Status int

//go:generate stringer -type=Status
const (
	// Unknown is initial status when none has been reported.
	Unknown Status = iota
	// Starting is status describing application is starting.
	Starting
	// Configuring is status describing application is configuring.
	Configuring
	// Running is status describing application is running.
	Running
	// Degraded is status describing application is degraded.
	Degraded
	// Failed is status describing application is failed. This status should
	// only be used in the case the beat should stop running as the failure
	// cannot be recovered.
	Failed
	// Stopping is status describing application is stopping.
	Stopping
)

// Namespace is the feature namespace for queue definition.
var Namespace = "libbeat.management"

// DebugK used as key for all things central management
var DebugK = "centralmgmt"

var centralMgmtKey = "x-pack-cm"

// StatusReporter provides a method to update current status of the beat.
type StatusReporter interface {
	// UpdateStatus called when the status of the beat has changed.
	UpdateStatus(status Status, msg string)
}

// Manager interacts with the beat to provide status updates and to receive
// configurations.
type Manager interface {
	StatusReporter

	// Enabled returns true if manager is enabled.
	Enabled() bool

	// Start the config manager giving it a stopFunc callback
	// so the beat can be told when to stop.
	Start(stopFunc func())

	// Stop the config manager.
	Stop()

	// CheckRawConfig check settings are correct before launching the beat.
	CheckRawConfig(cfg *common.Config) error

	// RegisterAction registers action handler with the client
	RegisterAction(action client.Action)
	// UnregisterAction unregisters action handler with the client
	UnregisterAction(action client.Action)

	// SetPayload sets the client payload
	SetPayload(map[string]interface{})
}

// PluginFunc for creating FactoryFunc if it matches a config
type PluginFunc func(*common.Config) FactoryFunc

// FactoryFunc for creating a config manager
type FactoryFunc func(*common.Config, *reload.Registry, uuid.UUID) (Manager, error)

// Register a config manager
func Register(name string, fn PluginFunc, stability feature.Stability) {
	f := feature.New(Namespace, name, fn, feature.MakeDetails(name, "", stability))
	feature.MustRegister(f)
}

// Factory retrieves config manager constructor. If no one is registered
// it will create a nil manager
func Factory(cfg *common.Config) FactoryFunc {
	factories, err := feature.GlobalRegistry().LookupAll(Namespace)
	if err != nil {
		return nilFactory
	}

	for _, f := range factories {
		if plugin, ok := f.Factory().(PluginFunc); ok {
			if factory := plugin(cfg); factory != nil {
				return factory
			}
		}
	}

	return nilFactory
}

type modeConfig struct {
	Mode string `config:"mode" yaml:"mode"`
}

func defaultModeConfig() *modeConfig {
	return &modeConfig{
		Mode: centralMgmtKey,
	}
}

// nilManager, fallback when no manager is present
type nilManager struct {
	logger *logp.Logger
	lock   sync.Mutex
	status Status
	msg    string
}

func nilFactory(*common.Config, *reload.Registry, uuid.UUID) (Manager, error) {
	log := logp.NewLogger("mgmt")
	return &nilManager{
		logger: log,
		status: Unknown,
		msg:    "",
	}, nil
}

func (*nilManager) Enabled() bool                           { return false }
func (*nilManager) Start(_ func())                          {}
func (*nilManager) Stop()                                   {}
func (*nilManager) CheckRawConfig(cfg *common.Config) error { return nil }
func (n *nilManager) UpdateStatus(status Status, msg string) {
	n.lock.Lock()
	defer n.lock.Unlock()
	if n.status != status || n.msg != msg {
		n.status = status
		n.msg = msg
		n.logger.Infof("Status change to %s: %s", status, msg)
	}
}

func (n *nilManager) RegisterAction(action client.Action) {}

func (n *nilManager) UnregisterAction(action client.Action) {}

func (n *nilManager) SetPayload(map[string]interface{}) {}
