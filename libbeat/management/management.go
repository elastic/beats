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

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/reload"
	"github.com/menderesk/beats/v7/libbeat/feature"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/elastic-agent-client/v7/pkg/client"
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

	// Start needs to invoked when the system is ready to receive an external configuration and
	// also ready to start ingesting new events. The manager expects that all the reloadable and
	// reloadable list are fixed for the whole lifetime of the manager.
	//
	// Notes: Adding dynamically new reloadable hooks at runtime can lead to inconsistency in the
	// execution.
	Start() error

	// Stop when this method is called, the manager will stop receiving new actions, no more action
	// will be propagated to the handlers and will not try to configure any reloadable parts.
	// When the manager is stop the callback will be called to signal that the system can terminate.
	//
	// Calls to 'CheckRawConfig()' or 'SetPayload()' will be ignored after calling stop.
	//
	// Note: Stop will not call 'UnregisterAction()' automaticallty.
	Stop()

	// SetStopCallback accepts a function that need to be called when the manager want to shutdown the
	// beats. This is needed when you want your beats to be gracefully shutdown remotely by the Elastic Agent
	// when a policy doesn't need to run this beat.
	SetStopCallback(f func())

	// CheckRawConfig check settings are correct before launching the beat.
	CheckRawConfig(cfg *common.Config) error

	// RegisterAction registers action handler with the client
	RegisterAction(action client.Action)

	// UnregisterAction unregisters action handler with the client
	UnregisterAction(action client.Action)

	// SetPayload Allows to add additional metadata to future requests made by the manager.
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
	logger   *logp.Logger
	lock     sync.Mutex
	status   Status
	msg      string
	stopFunc func()
}

func nilFactory(*common.Config, *reload.Registry, uuid.UUID) (Manager, error) {
	log := logp.NewLogger("mgmt")
	return &nilManager{
		logger: log,
		status: Unknown,
		msg:    "",
	}, nil
}

func (*nilManager) SetStopCallback(func())                  {}
func (*nilManager) Enabled() bool                           { return false }
func (*nilManager) Start() error                            { return nil }
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
