// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/kibana"
)

const (
	apiStatusTimeout = 15 * time.Second
)

type clientSetter interface {
	SetClient(clienter)
}

type handlerPolicyChange struct {
	log       *logger.Logger
	emitter   emitterFunc
	agentInfo *info.AgentInfo
	config    *configuration.Configuration
	store     storage.Store
	setters   []clientSetter
}

func (h *handlerPolicyChange) Handle(ctx context.Context, a action, acker fleetAcker) error {
	h.log.Debugf("handlerPolicyChange: action '%+v' received", a)
	action, ok := a.(*fleetapi.ActionPolicyChange)
	if !ok {
		return fmt.Errorf("invalid type, expected ActionPolicyChange and received %T", a)
	}

	c, err := config.NewConfigFrom(action.Policy)
	if err != nil {
		return errors.New(err, "could not parse the configuration from the policy", errors.TypeConfig)
	}

	h.log.Debugf("handlerPolicyChange: emit configuration for action %+v", a)
	err = h.handleKibanaHosts(ctx, c)
	if err != nil {
		return err
	}
	if err := h.emitter(c); err != nil {
		return err
	}

	return acker.Ack(ctx, action)
}

func (h *handlerPolicyChange) handleKibanaHosts(ctx context.Context, c *config.Config) (err error) {
	// do not update kibana host from policy; no setters provided with local Fleet Server
	if len(h.setters) == 0 {
		return nil
	}

	cfg, err := configuration.NewFromConfig(c)
	if err != nil {
		return errors.New(err, "could not parse the configuration from the policy", errors.TypeConfig)
	}
	if kibanaEqual(h.config.Fleet.Kibana, cfg.Fleet.Kibana) {
		// already the same hosts
		return nil
	}

	// only set protocol/hosts as that is all Fleet currently sends
	prevProtocol := h.config.Fleet.Kibana.Protocol
	prevPath := h.config.Fleet.Kibana.Path
	prevHosts := h.config.Fleet.Kibana.Hosts
	h.config.Fleet.Kibana.Protocol = cfg.Fleet.Kibana.Protocol
	h.config.Fleet.Kibana.Path = cfg.Fleet.Kibana.Path
	h.config.Fleet.Kibana.Hosts = cfg.Fleet.Kibana.Hosts

	// rollback on failure
	defer func() {
		if err != nil {
			h.config.Fleet.Kibana.Protocol = prevProtocol
			h.config.Fleet.Kibana.Path = prevPath
			h.config.Fleet.Kibana.Hosts = prevHosts
		}
	}()

	client, err := fleetapi.NewAuthWithConfig(h.log, h.config.Fleet.AccessAPIKey, h.config.Fleet.Kibana)
	if err != nil {
		return errors.New(
			err, "fail to create API client with updated hosts",
			errors.TypeNetwork, errors.M("hosts", h.config.Fleet.Kibana.Hosts))
	}
	ctx, cancel := context.WithTimeout(ctx, apiStatusTimeout)
	defer cancel()
	_, err = client.Send(ctx, "GET", "/api/status", nil, nil, nil)
	if err != nil {
		return errors.New(
			err, "fail to communicate with updated API client hosts",
			errors.TypeNetwork, errors.M("hosts", h.config.Fleet.Kibana.Hosts))
	}
	reader, err := fleetToReader(h.agentInfo, h.config)
	if err != nil {
		return errors.New(
			err, "fail to persist updated API client hosts",
			errors.TypeUnexpected, errors.M("hosts", h.config.Fleet.Kibana.Hosts))
	}
	err = h.store.Save(reader)
	if err != nil {
		return errors.New(
			err, "fail to persist updated API client hosts",
			errors.TypeFilesystem, errors.M("hosts", h.config.Fleet.Kibana.Hosts))
	}
	for _, setter := range h.setters {
		setter.SetClient(client)
	}
	return nil
}

func kibanaEqual(k1 *kibana.Config, k2 *kibana.Config) bool {
	if k1.Protocol != k2.Protocol {
		return false
	}
	if k1.Path != k2.Path {
		return false
	}

	sort.Strings(k1.Hosts)
	sort.Strings(k2.Hosts)
	if len(k1.Hosts) != len(k2.Hosts) {
		return false
	}
	for i, v := range k1.Hosts {
		if v != k2.Hosts[i] {
			return false
		}
	}
	return true
}

func fleetToReader(agentInfo *info.AgentInfo, cfg *configuration.Configuration) (io.Reader, error) {
	configToStore := map[string]interface{}{
		"fleet": cfg.Fleet,
		"agent": map[string]interface{}{
			"id": agentInfo.AgentID(),
		},
	}
	data, err := yaml.Marshal(configToStore)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}
