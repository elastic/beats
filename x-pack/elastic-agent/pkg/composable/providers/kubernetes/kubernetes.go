// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kubernetes

import (
	"fmt"

	k8s "k8s.io/client-go/kubernetes"

	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

const (
	// NodePriority is the priority that node mappings are added to the provider.
	NodePriority = 0
	// PodPriority is the priority that pod mappings are added to the provider.
	PodPriority = 1
	// ContainerPriority is the priority that container mappings are added to the provider.
	ContainerPriority = 2
	// ServicePriority is the priority that service mappings are added to the provider.
	ServicePriority = 3
)

func init() {
	composable.Providers.AddDynamicProvider("kubernetes", DynamicProviderBuilder)
}

type dynamicProvider struct {
	logger *logger.Logger
	config *Config
}

// DynamicProviderBuilder builds the dynamic provider.
func DynamicProviderBuilder(logger *logger.Logger, c *config.Config) (composable.DynamicProvider, error) {
	var cfg Config
	if c == nil {
		c = config.New()
	}
	err := c.Unpack(&cfg)
	if err != nil {
		return nil, errors.New(err, "failed to unpack configuration")
	}
	return &dynamicProvider{logger, &cfg}, nil
}

// Run runs the kubernetes context provider.
func (p *dynamicProvider) Run(comm composable.DynamicProviderComm) error {
	if p.config.Resources.Pod.Enabled {
		err := p.watchResource(comm, "pod", p.config)
		if err != nil {
			return err
		}
	}
	if p.config.Resources.Node.Enabled {
		err := p.watchResource(comm, "node", p.config)
		if err != nil {
			return err
		}
	}
	if p.config.Resources.Service.Enabled {
		err := p.watchResource(comm, "service", p.config)
		if err != nil {
			return err
		}
	}
	return nil
}

// watchResource initializes the proper watcher according to the given resource (pod, node, service)
// and starts watching for such resource's events.
func (p *dynamicProvider) watchResource(
	comm composable.DynamicProviderComm,
	resourceType string,
	config *Config) error {
	client, err := kubernetes.GetKubernetesClient(config.KubeConfig)
	if err != nil {
		// info only; return nil (do nothing)
		p.logger.Debugf("Kubernetes provider for resource %s skipped, unable to connect: %s", resourceType, err)
		return nil
	}

	// Ensure that node is set correctly whenever the scope is set to "node". Make sure that node is empty
	// when cluster scope is enforced.
	p.logger.Infof("Kubernetes provider started for resource %s with %s scope", resourceType, p.config.Scope)
	if p.config.Scope == "node" {
		p.logger.Debugf(
			"Initializing Kubernetes watcher for resource %s using node: %v",
			resourceType,
			config.Node)
		config.Node = kubernetes.DiscoverKubernetesNode(
			p.logger, config.Node,
			kubernetes.IsInCluster(config.KubeConfig),
			client)
	} else {
		config.Node = ""
	}

	watcher, err := p.newWatcher(resourceType, comm, client, config)
	if err != nil {
		return errors.New(err, "couldn't create kubernetes watcher for resource %s", resourceType)
	}

	err = watcher.Start()
	if err != nil {
		return errors.New(err, "couldn't start kubernetes watcher for resource %s", resourceType)
	}
	return nil
}

// newWatcher initializes the proper watcher according to the given resource (pod, node, service).
func (p *dynamicProvider) newWatcher(
	resourceType string,
	comm composable.DynamicProviderComm,
	client k8s.Interface,
	config *Config) (kubernetes.Watcher, error) {
	switch resourceType {
	case "pod":
		watcher, err := NewPodWatcher(comm, config, p.logger, client, p.config.Scope)
		if err != nil {
			return nil, err
		}
		return watcher, nil
	case "node":
		watcher, err := NewNodeWatcher(comm, config, p.logger, client, p.config.Scope)
		if err != nil {
			return nil, err
		}
		return watcher, nil
	case "service":
		watcher, err := NewServiceWatcher(comm, config, p.logger, client, p.config.Scope)
		if err != nil {
			return nil, err
		}
		return watcher, nil
	default:
		return nil, fmt.Errorf("unsupported autodiscover resource %s", resourceType)
	}
}
