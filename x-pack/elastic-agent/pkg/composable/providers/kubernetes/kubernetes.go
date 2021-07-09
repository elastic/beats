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
	// PodPriority is the priority that pod mappings are added to the provider.
	PodPriority = 0
	// ContainerPriority is the priority that container mappings are added to the provider.
	ContainerPriority = 1
	// NodePriority is the priority that node mappings are added to the provider.
	NodePriority = 0
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

// Run runs the environment context provider.
func (p *dynamicProvider) Run(comm composable.DynamicProviderComm) error {
	client, err := kubernetes.GetKubernetesClient(p.config.KubeConfig)
	if err != nil {
		// info only; return nil (do nothing)
		p.logger.Debugf("Kubernetes provider skipped, unable to connect: %s", err)
		return nil
	}

	// Ensure that node is set correctly whenever the scope is set to "node". Make sure that node is empty
	// when cluster scope is enforced.
	p.logger.Infof("Kubernetes provider started with %s scope", p.config.Scope)
	if p.config.Scope == "node" {
		p.logger.Debugf("Initializing Kubernetes watcher using node: %v", p.config.Node)
		p.config.Node = kubernetes.DiscoverKubernetesNode(p.logger, p.config.Node, kubernetes.IsInCluster(p.config.KubeConfig), client)
	} else {
		p.config.Node = ""
	}

	watcher, err := p.newWatcher(comm, client)
	if err != nil {
		return errors.New(err, "couldn't create kubernetes watcher")
	}

	err = watcher.Start()
	if err != nil {
		return errors.New(err, "couldn't start kubernetes watcher")
	}

	return nil
}

// newWatcher initializes the proper watcher according to the given resource (pod, node, service).
func (p *dynamicProvider) newWatcher(comm composable.DynamicProviderComm, client k8s.Interface) (kubernetes.Watcher, error) {
	switch p.config.Resource {
	case "pod":
		watcher, err := NewPodWatcher(comm, p.config, p.logger, client)
		if err != nil {
			return nil, err
		}
		return watcher, nil
	case "node":
		watcher, err := NewNodeWatcher(comm, p.config, p.logger, client)
		if err != nil {
			return nil, err
		}
		return watcher, nil
	case "service":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported autodiscover resource %s", p.config.Resource)
	}
}
