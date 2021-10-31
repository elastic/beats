// Copyright 2018 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

// Package status implements status reporting.
package status

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/pkg/errors"

	"github.com/open-policy-agent/opa/logging"
	"github.com/open-policy-agent/opa/metrics"
	"github.com/open-policy-agent/opa/plugins"
	"github.com/open-policy-agent/opa/plugins/bundle"
	"github.com/open-policy-agent/opa/util"
)

// Logger defines the interface for status plugins.
type Logger interface {
	plugins.Plugin

	Log(context.Context, *UpdateRequestV1) error
}

// UpdateRequestV1 represents the status update message that OPA sends to
// remote HTTP endpoints.
type UpdateRequestV1 struct {
	Labels    map[string]string          `json:"labels"`
	Bundle    *bundle.Status             `json:"bundle,omitempty"` // Deprecated: Use bulk `bundles` status updates instead
	Bundles   map[string]*bundle.Status  `json:"bundles,omitempty"`
	Discovery *bundle.Status             `json:"discovery,omitempty"`
	Metrics   map[string]interface{}     `json:"metrics,omitempty"`
	Plugins   map[string]*plugins.Status `json:"plugins,omitempty"`
}

// Plugin implements status reporting. Updates can be triggered by the caller.
type Plugin struct {
	manager          *plugins.Manager
	config           Config
	bundleCh         chan bundle.Status // Deprecated: Use bulk bundle status updates instead
	lastBundleStatus *bundle.Status     // Deprecated: Use bulk bundle status updates instead

	bulkBundleCh       chan map[string]*bundle.Status
	lastBundleStatuses map[string]*bundle.Status

	discoCh         chan bundle.Status
	lastDiscoStatus *bundle.Status

	pluginStatusCh     chan map[string]*plugins.Status
	lastPluginStatuses map[string]*plugins.Status

	stop     chan chan struct{}
	reconfig chan interface{}
	metrics  metrics.Metrics
	logger   logging.Logger
	trigger  chan trigger
}

// Config contains configuration for the plugin.
type Config struct {
	Plugin        *string              `json:"plugin"`
	Service       string               `json:"service"`
	PartitionName string               `json:"partition_name,omitempty"`
	ConsoleLogs   bool                 `json:"console"`
	Trigger       *plugins.TriggerMode `json:"trigger,omitempty"` // trigger mode
}

type trigger struct {
	ctx  context.Context
	done chan error
}

func (c *Config) validateAndInjectDefaults(services []string, pluginsList []string, trigger *plugins.TriggerMode) error {

	if c.Plugin != nil {
		var found bool
		for _, other := range pluginsList {
			if other == *c.Plugin {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid plugin name %q in status", *c.Plugin)
		}
	} else if c.Service == "" && len(services) != 0 && !c.ConsoleLogs {
		// For backwards compatibility allow defaulting to the first
		// service listed, but only if console logging is disabled. If enabled
		// we can't tell if the deployer wanted to use only console logs or
		// both console logs and the default service option.
		c.Service = services[0]
	} else if c.Service != "" {
		found := false

		for _, svc := range services {
			if svc == c.Service {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("invalid service name %q in status", c.Service)
		}
	}

	// If a plugin or service wasn't found, and console logging isn't enabled.
	if c.Plugin == nil && c.Service == "" && !c.ConsoleLogs {
		return fmt.Errorf("invalid status config, must have a `service`, `plugin`, or `console` logging specified")
	}

	t, err := plugins.ValidateAndInjectDefaultsForTriggerMode(trigger, c.Trigger)
	if err != nil {
		return errors.Wrap(err, "invalid status config")
	}
	c.Trigger = t

	return nil
}

// ParseConfig validates the config and injects default values.
func ParseConfig(config []byte, services []string, pluginsList []string) (*Config, error) {
	t := plugins.DefaultTriggerMode
	return NewConfigBuilder().WithBytes(config).WithServices(services).WithPlugins(pluginsList).WithTriggerMode(&t).Parse()
}

// ConfigBuilder assists in the construction of the plugin configuration.
type ConfigBuilder struct {
	raw      []byte
	services []string
	plugins  []string
	trigger  *plugins.TriggerMode
}

// NewConfigBuilder returns a new ConfigBuilder to build and parse the plugin config.
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{}
}

// WithBytes sets the raw plugin config.
func (b *ConfigBuilder) WithBytes(config []byte) *ConfigBuilder {
	b.raw = config
	return b
}

// WithServices sets the services that implement control plane APIs.
func (b *ConfigBuilder) WithServices(services []string) *ConfigBuilder {
	b.services = services
	return b
}

// WithPlugins sets the list of named plugins for status updates.
func (b *ConfigBuilder) WithPlugins(plugins []string) *ConfigBuilder {
	b.plugins = plugins
	return b
}

// WithTriggerMode sets the plugin trigger mode.
func (b *ConfigBuilder) WithTriggerMode(trigger *plugins.TriggerMode) *ConfigBuilder {
	b.trigger = trigger
	return b
}

// Parse validates the config and injects default values.
func (b *ConfigBuilder) Parse() (*Config, error) {
	if b.raw == nil {
		return nil, nil
	}

	var parsedConfig Config

	if err := util.Unmarshal(b.raw, &parsedConfig); err != nil {
		return nil, err
	}

	if err := parsedConfig.validateAndInjectDefaults(b.services, b.plugins, b.trigger); err != nil {
		return nil, err
	}

	return &parsedConfig, nil
}

// New returns a new Plugin with the given config.
func New(parsedConfig *Config, manager *plugins.Manager) *Plugin {
	p := &Plugin{
		manager:        manager,
		config:         *parsedConfig,
		bundleCh:       make(chan bundle.Status),
		bulkBundleCh:   make(chan map[string]*bundle.Status),
		discoCh:        make(chan bundle.Status),
		stop:           make(chan chan struct{}),
		reconfig:       make(chan interface{}),
		pluginStatusCh: make(chan map[string]*plugins.Status),
		logger:         manager.Logger().WithFields(map[string]interface{}{"plugin": Name}),
		trigger:        make(chan trigger),
	}

	p.manager.UpdatePluginStatus(Name, &plugins.Status{State: plugins.StateNotReady})

	return p
}

// WithMetrics sets the global metrics provider to be used by the plugin.
func (p *Plugin) WithMetrics(m metrics.Metrics) *Plugin {
	p.metrics = m
	return p
}

// Name identifies the plugin on manager.
const Name = "status"

// Lookup returns the status plugin registered with the manager.
func Lookup(manager *plugins.Manager) *Plugin {
	if p := manager.Plugin(Name); p != nil {
		return p.(*Plugin)
	}
	return nil
}

// Start starts the plugin.
func (p *Plugin) Start(ctx context.Context) error {
	p.logger.Info("Starting status reporter.")

	go p.loop()

	// Setup a listener for plugin statuses, but only after starting the loop
	// to prevent blocking threads pushing the plugin updates.
	p.manager.RegisterPluginStatusListener(Name, p.UpdatePluginStatus)

	// Set the status plugin's status to OK now that everything is registered and
	// the loop is running. This will trigger an update on the listener with the
	// current status of all the other plugins too.
	p.manager.UpdatePluginStatus(Name, &plugins.Status{State: plugins.StateOK})
	return nil
}

// Stop stops the plugin.
func (p *Plugin) Stop(ctx context.Context) {
	p.logger.Info("Stopping status reporter.")
	p.manager.UnregisterPluginStatusListener(Name)
	done := make(chan struct{})
	p.stop <- done
	<-done
	p.manager.UpdatePluginStatus(Name, &plugins.Status{State: plugins.StateNotReady})
}

// UpdateBundleStatus notifies the plugin that the policy bundle was updated.
// Deprecated: Use BulkUpdateBundleStatus instead.
func (p *Plugin) UpdateBundleStatus(status bundle.Status) {
	p.bundleCh <- status
}

// BulkUpdateBundleStatus notifies the plugin that the policy bundle was updated.
func (p *Plugin) BulkUpdateBundleStatus(status map[string]*bundle.Status) {
	p.bulkBundleCh <- status
}

// UpdateDiscoveryStatus notifies the plugin that the discovery bundle was updated.
func (p *Plugin) UpdateDiscoveryStatus(status bundle.Status) {
	p.discoCh <- status
}

// UpdatePluginStatus notifies the plugin that a plugin status was updated.
func (p *Plugin) UpdatePluginStatus(status map[string]*plugins.Status) {
	p.pluginStatusCh <- status
}

// Reconfigure notifies the plugin with a new configuration.
func (p *Plugin) Reconfigure(_ context.Context, config interface{}) {
	p.reconfig <- config
}

// Trigger can be used to control when the plugin attempts to upload
//status in manual triggering mode.
func (p *Plugin) Trigger(ctx context.Context) error {
	done := make(chan error)
	p.trigger <- trigger{ctx: ctx, done: done}

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Plugin) loop() {

	ctx, cancel := context.WithCancel(context.Background())

	for {

		select {
		case statuses := <-p.pluginStatusCh:
			p.lastPluginStatuses = statuses
			if *p.config.Trigger == plugins.TriggerPeriodic {
				err := p.oneShot(ctx)
				if err != nil {
					p.logger.Error("%v.", err)
				} else {
					p.logger.Info("Status update sent successfully in response to plugin update.")
				}
			}

		case statuses := <-p.bulkBundleCh:
			p.lastBundleStatuses = statuses
			if *p.config.Trigger == plugins.TriggerPeriodic {
				err := p.oneShot(ctx)
				if err != nil {
					p.logger.Error("%v.", err)
				} else {
					p.logger.Info("Status update sent successfully in response to bundle update.")
				}
			}

		case status := <-p.bundleCh:
			p.lastBundleStatus = &status
			err := p.oneShot(ctx)
			if err != nil {
				p.logger.Error("%v.", err)
			} else {
				p.logger.Info("Status update sent successfully in response to bundle update.")
			}
		case status := <-p.discoCh:
			p.lastDiscoStatus = &status
			if *p.config.Trigger == plugins.TriggerPeriodic {
				err := p.oneShot(ctx)
				if err != nil {
					p.logger.Error("%v.", err)
				} else {
					p.logger.Info("Status update sent successfully in response to discovery update.")
				}
			}
		case newConfig := <-p.reconfig:
			p.reconfigure(newConfig)
		case update := <-p.trigger:
			err := p.oneShot(update.ctx)
			if err != nil {
				p.logger.Error("%v.", err)
				if update.ctx.Err() == nil {
					update.done <- err
				}
			} else {
				p.logger.Info("Status update sent successfully in response to manual trigger.")
			}
			close(update.done)
		case done := <-p.stop:
			cancel()
			done <- struct{}{}
			return
		}
	}
}

func (p *Plugin) oneShot(ctx context.Context) error {

	req := &UpdateRequestV1{
		Labels:    p.manager.Labels(),
		Discovery: p.lastDiscoStatus,
		Bundle:    p.lastBundleStatus,
		Bundles:   p.lastBundleStatuses,
		Plugins:   p.lastPluginStatuses,
	}

	if p.metrics != nil {
		req.Metrics = map[string]interface{}{p.metrics.Info().Name: p.metrics.All()}
	}

	if p.config.ConsoleLogs {
		err := p.logUpdate(req)
		if err != nil {
			p.logger.Error("Failed to log to console: %v.", err)
		}
	}

	if p.config.Plugin != nil {
		proxy, ok := p.manager.Plugin(*p.config.Plugin).(Logger)
		if !ok {
			return fmt.Errorf("plugin does not implement Logger interface")
		}
		return proxy.Log(ctx, req)
	}

	if p.config.Service != "" {
		resp, err := p.manager.Client(p.config.Service).
			WithJSON(req).
			Do(ctx, "POST", fmt.Sprintf("/status/%v", p.config.PartitionName))

		if err != nil {
			return errors.Wrap(err, "Status update failed")
		}

		defer util.Close(resp)

		switch resp.StatusCode {
		case http.StatusOK:
			return nil
		case http.StatusNotFound:
			return fmt.Errorf("status update failed, server replied with not found")
		case http.StatusUnauthorized:
			return fmt.Errorf("status update failed, server replied with not authorized")
		default:
			return fmt.Errorf("status update failed, server replied with HTTP %v", resp.StatusCode)
		}
	}
	return nil
}

func (p *Plugin) reconfigure(config interface{}) {
	newConfig := config.(*Config)

	if reflect.DeepEqual(p.config, *newConfig) {
		p.logger.Debug("Status reporter configuration unchanged.")
		return
	}

	p.logger.Info("Status reporter configuration changed.")
	p.config = *newConfig
}

func (p *Plugin) logUpdate(update *UpdateRequestV1) error {
	eventBuf, err := json.Marshal(&update)
	if err != nil {
		return err
	}
	fields := map[string]interface{}{}
	err = util.UnmarshalJSON(eventBuf, &fields)
	if err != nil {
		return err
	}
	p.manager.ConsoleLogger().WithFields(fields).WithFields(map[string]interface{}{
		"type": "openpolicyagent.org/status",
	}).Info("Status Log")
	return nil
}
