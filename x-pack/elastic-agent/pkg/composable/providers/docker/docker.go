// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package docker

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/bus"
	"github.com/elastic/beats/v7/libbeat/common/docker"
	"github.com/elastic/beats/v7/libbeat/common/safemapstr"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

// ContainerPriority is the priority that container mappings are added to the provider.
const ContainerPriority = 0

func init() {
	composable.Providers.AddDynamicProvider("docker", DynamicProviderBuilder)
}

type dockerContainerData struct {
	container  *docker.Container
	mapping    map[string]interface{}
	processors []map[string]interface{}
}
type dynamicProvider struct {
	logger *logger.Logger
	config *Config
}

// Run runs the environment context provider.
func (c *dynamicProvider) Run(comm composable.DynamicProviderComm) error {
	watcher, err := docker.NewWatcher(c.logger, c.config.Host, c.config.TLS, false)
	if err != nil {
		// info only; return nil (do nothing)
		c.logger.Infof("Docker provider skipped, unable to connect: %s", err)
		return nil
	}
	startListener := watcher.ListenStart()
	stopListener := watcher.ListenStop()
	stoppers := map[string]*time.Timer{}
	stopTrigger := make(chan *dockerContainerData)

	if err := watcher.Start(); err != nil {
		// info only; return nil (do nothing)
		c.logger.Infof("Docker provider skipped, unable to connect: %s", err)
		return nil
	}

	go func() {
		for {
			select {
			case <-comm.Done():
				startListener.Stop()
				stopListener.Stop()

				// Stop all timers before closing the channel
				for _, stopper := range stoppers {
					stopper.Stop()
				}
				close(stopTrigger)
				return
			case event := <-startListener.Events():
				data, err := generateData(event)
				if err != nil {
					c.logger.Errorf("%s", err)
					continue
				}
				if stopper, ok := stoppers[data.container.ID]; ok {
					c.logger.Debugf("container %s is restarting, aborting pending stop", data.container.ID)
					stopper.Stop()
					delete(stoppers, data.container.ID)
					return
				}
				comm.AddOrUpdate(data.container.ID, ContainerPriority, data.mapping, data.processors)
			case event := <-stopListener.Events():
				data, err := generateData(event)
				if err != nil {
					c.logger.Errorf("%s", err)
					continue
				}
				stopper := time.AfterFunc(c.config.CleanupTimeout, func() {
					stopTrigger <- data
				})
				stoppers[data.container.ID] = stopper
			case data := <-stopTrigger:
				delete(stoppers, data.container.ID)
				comm.Remove(data.container.ID)
			}
		}
	}()

	return nil
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

func generateData(event bus.Event) (*dockerContainerData, error) {
	container, ok := event["container"].(*docker.Container)
	if !ok {
		return nil, fmt.Errorf("unable to get container from watcher event")
	}

	labelMap := common.MapStr{}
	processorLabelMap := common.MapStr{}
	for k, v := range container.Labels {
		safemapstr.Put(labelMap, k, v)
		processorLabelMap.Put(common.DeDot(k), v)
	}

	data := &dockerContainerData{
		container: container,
		mapping: map[string]interface{}{
			"container": map[string]interface{}{
				"id":     container.ID,
				"name":   container.Name,
				"image":  container.Image,
				"labels": labelMap,
			},
		},
		processors: []map[string]interface{}{
			{
				"add_fields": map[string]interface{}{
					"fields": map[string]interface{}{
						"id":     container.ID,
						"name":   container.Name,
						"image":  container.Image,
						"labels": processorLabelMap,
					},
					"target": "container",
				},
			},
		},
	}
	return data, nil
}
