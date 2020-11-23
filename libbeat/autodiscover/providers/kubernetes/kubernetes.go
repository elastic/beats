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

// +build linux darwin windows

package kubernetes

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/autodiscover/template"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/bus"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes/k8skeystore"
	"github.com/elastic/beats/v7/libbeat/keystore"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func init() {
	autodiscover.Registry.AddProvider("kubernetes", AutodiscoverBuilder)
}

// Eventer allows defining ways in which kubernetes resource events are observed and processed
type Eventer interface {
	kubernetes.ResourceEventHandler
	GenerateHints(event bus.Event) bus.Event
	Start() error
	Stop()
}

// EventManager allows defining ways in which kubernetes resource events are observed and processed
type EventManager interface {
	GenerateHints(event bus.Event) bus.Event
	Start()
	Stop()
}

// Provider implements autodiscover provider for docker containers
type Provider struct {
	config       *Config
	bus          bus.Bus
	templates    template.Mapper
	builders     autodiscover.Builders
	appenders    autodiscover.Appenders
	logger       *logp.Logger
	eventManager EventManager
}

// eventerManager implements start/stop methods for autodiscover provider with resource eventer
type eventerManager struct {
	eventer Eventer
	logger  *logp.Logger
}

// leaderElectionManager implements start/stop methods for autodiscover provider with leaderElection
type leaderElectionManager struct {
	leaderElection       leaderelection.LeaderElectionConfig
	cancelLeaderElection context.CancelFunc
	logger               *logp.Logger
}

// AutodiscoverBuilder builds and returns an autodiscover provider
func AutodiscoverBuilder(
	beatName string,
	bus bus.Bus,
	uuid uuid.UUID,
	c *common.Config,
	keystore keystore.Keystore,
) (autodiscover.Provider, error) {
	logger := logp.NewLogger("autodiscover")

	errWrap := func(err error) error {
		return errors.Wrap(err, "error setting up kubernetes autodiscover provider")
	}

	config := defaultConfig()
	config.LeaderLease = fmt.Sprintf("%v-cluster-leader", beatName)
	err := c.Unpack(&config)
	if err != nil {
		return nil, errWrap(err)
	}

	client, err := kubernetes.GetKubernetesClient(config.KubeConfig)
	if err != nil {
		return nil, errWrap(err)
	}

	k8sKeystoreProvider := k8skeystore.NewKubernetesKeystoresRegistry(logger, client)

	mapper, err := template.NewConfigMapper(config.Templates, keystore, k8sKeystoreProvider)
	if err != nil {
		return nil, errWrap(err)
	}

	builders, err := autodiscover.NewBuilders(config.Builders, config.Hints, k8sKeystoreProvider)
	if err != nil {
		return nil, errWrap(err)
	}

	appenders, err := autodiscover.NewAppenders(config.Appenders)
	if err != nil {
		return nil, errWrap(err)
	}

	p := &Provider{
		config:    config,
		bus:       bus,
		templates: mapper,
		builders:  builders,
		appenders: appenders,
		logger:    logger,
	}

	if p.config.Unique {
		p.eventManager, err = NewLeaderElectionManager(uuid, config, client, p.startLeading, p.stopLeading, logger)
	} else {
		p.eventManager, err = NewEventerManager(uuid, c, config, client, p.publish)
	}

	if err != nil {
		return nil, errWrap(err)
	}

	return p, nil
}

// Start for Runner interface.
func (p *Provider) Start() {
	p.eventManager.Start()
}

// Stop signals the stop channel to force the watch loop routine to stop.
func (p *Provider) Stop() {
	p.eventManager.Stop()
}

// String returns a description of kubernetes autodiscover provider.
func (p *Provider) String() string {
	return "kubernetes"
}

func (p *Provider) publish(event bus.Event) {
	// Try to match a config
	if config := p.templates.GetConfig(event); config != nil {
		event["config"] = config
	} else {
		// If there isn't a default template then attempt to use builders
		e := p.eventManager.GenerateHints(event)
		if config := p.builders.GetConfig(e); config != nil {
			event["config"] = config
		}
	}

	// Call all appenders to append any extra configuration
	p.appenders.Append(event)
	p.bus.Publish(event)
}

func (p *Provider) startLeading(uuid string, eventID string) {
	event := bus.Event{
		"start":    true,
		"provider": uuid,
		"id":       eventID,
		"unique":   "true",
	}
	if config := p.templates.GetConfig(event); config != nil {
		event["config"] = config
	}
	p.bus.Publish(event)
}

func (p *Provider) stopLeading(uuid string, eventID string) {
	event := bus.Event{
		"stop":     true,
		"provider": uuid,
		"id":       eventID,
		"unique":   "true",
	}
	if config := p.templates.GetConfig(event); config != nil {
		event["config"] = config
	}
	p.bus.Publish(event)
}

func NewEventerManager(
	uuid uuid.UUID,
	c *common.Config,
	cfg *Config,
	client k8s.Interface,
	publish func(event bus.Event),
) (EventManager, error) {
	var err error
	em := &eventerManager{}
	switch cfg.Resource {
	case "pod":
		em.eventer, err = NewPodEventer(uuid, c, client, publish)
	case "node":
		em.eventer, err = NewNodeEventer(uuid, c, client, publish)
	case "service":
		em.eventer, err = NewServiceEventer(uuid, c, client, publish)
	default:
		return nil, fmt.Errorf("unsupported autodiscover resource %s", cfg.Resource)
	}

	if err != nil {
		return nil, err
	}
	return em, nil
}

func NewLeaderElectionManager(
	uuid uuid.UUID,
	cfg *Config,
	client k8s.Interface,
	startLeading func(uuid string, eventID string),
	stopLeading func(uuid string, eventID string),
	logger *logp.Logger,
) (EventManager, error) {
	lem := &leaderElectionManager{logger: logger}
	var id string
	if cfg.Node != "" {
		id = "beats-leader-" + cfg.Node
	} else {
		id = "beats-leader-" + uuid.String()
	}
	ns, err := kubernetes.InClusterNamespace()
	if err != nil {
		ns = "default"
	}
	lease := metav1.ObjectMeta{
		Name:      cfg.LeaderLease,
		Namespace: ns,
	}
	metaUID := lease.GetObjectMeta().GetUID()
	lem.leaderElection = leaderelection.LeaderElectionConfig{
		Lock: &resourcelock.LeaseLock{
			LeaseMeta: lease,
			Client:    client.CoordinationV1(),
			LockConfig: resourcelock.ResourceLockConfig{
				Identity: id,
			},
		},
		ReleaseOnCancel: false,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				logger.Debugf("leader election lock GAINED, id %v", id)
				eventID := fmt.Sprintf("%v-%v", metaUID, time.Now().UnixNano())
				startLeading(uuid.String(), eventID)
			},
			OnStoppedLeading: func() {
				logger.Debugf("leader election lock LOST, id %v", id)
				eventID := fmt.Sprintf("%v-%v", metaUID, time.Now().UnixNano())
				stopLeading(uuid.String(), eventID)
			},
		},
	}
	return lem, nil
}

// Start for EventManager interface.
func (p *eventerManager) Start() {
	if err := p.eventer.Start(); err != nil {
		p.logger.Errorf("Error starting kubernetes autodiscover provider: %s", err)
	}
}

// Stop signals the stop channel to force the watch loop routine to stop.
func (p *eventerManager) Stop() {
	p.eventer.Stop()
}

// GenerateHints for EventManager interface.
func (p *eventerManager) GenerateHints(event bus.Event) bus.Event {
	return p.eventer.GenerateHints(event)
}

// Start for EventManager interface.
func (p *leaderElectionManager) Start() {
	ctx, cancel := context.WithCancel(context.TODO())
	p.cancelLeaderElection = cancel
	p.startLeaderElector(ctx, p.leaderElection)
}

// Stop signals the stop channel to force the leader election loop routine to stop.
func (p *leaderElectionManager) Stop() {
	if p.cancelLeaderElection != nil {
		p.cancelLeaderElection()
	}
}

// GenerateHints for EventManager interface.
func (p *leaderElectionManager) GenerateHints(event bus.Event) bus.Event {
	return event
}

// startLeaderElector starts a Leader Elector in the background with the provided config
func (p *leaderElectionManager) startLeaderElector(ctx context.Context, lec leaderelection.LeaderElectionConfig) {
	le, err := leaderelection.NewLeaderElector(lec)
	if err != nil {
		p.logger.Errorf("error while creating Leader Elector: %v", err)
	}
	p.logger.Debugf("Starting Leader Elector")
	go le.Run(ctx)
}
