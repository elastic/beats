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
	if p.config.Resource.Pod != nil {
		resourceConfig := p.config.Resource.Pod
		err := p.watchResource(comm, "pod", resourceConfig)
		if err != nil {
			return err
		}
	}
	if p.config.Resource.Node != nil {
		resourceConfig := p.config.Resource.Node
		err := p.watchResource(comm, "node", resourceConfig)
		if err != nil {
			return err
		}
	}
	if p.config.Resource.Service != nil {
		resourceConfig := p.config.Resource.Service
		err := p.watchResource(comm, "service", resourceConfig)
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
	resourceConfig *ResourceConfig) error {
	client, err := kubernetes.GetKubernetesClient(resourceConfig.KubeConfig)
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
			resourceConfig.Node)
		resourceConfig.Node = kubernetes.DiscoverKubernetesNode(
			p.logger, resourceConfig.Node,
			kubernetes.IsInCluster(resourceConfig.KubeConfig),
			client)
	} else {
		resourceConfig.Node = ""
	}

	watcher, err := p.newWatcher(resourceType, comm, client, resourceConfig)
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
	config *ResourceConfig) (kubernetes.Watcher, error) {
	switch resourceType {
	case "pod":
		watcher, err := NewPodWatcher(comm, config, p.logger, client)
		if err != nil {
			return nil, err
		}
		return watcher, nil
	case "node":
		watcher, err := NewNodeWatcher(comm, config, p.logger, client)
		if err != nil {
			return nil, err
		}
		return watcher, nil
	case "service":
		watcher, err := NewServiceWatcher(comm, config, p.logger, client)
		if err != nil {
			return nil, err
		}
		return watcher, nil
	default:
		return nil, fmt.Errorf("unsupported autodiscover resource %s", resourceType)
	}
}
