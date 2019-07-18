// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"fmt"
	"time"

	"github.com/elastic/beats/x-pack/functionbeat/config"
)

// FunctionConfig stores the configuration of a Google Cloud Function
type FunctionConfig struct {
	Description         string                 `config:"description"`
	MemorySize          config.MemSizeFactor64 `config:"memory_size"`
	Timeout             time.Duration          `config:"timeout" validate:"nonzero,positive"`
	ServiceAccountEmail string                 `config:"service_account_email"`
	Labels              map[string]string      `config:"labels"`
	VPCConnector        string                 `config:"vpc_connector"`
	MaxInstances        int                    `config:"maximum_instances"`
	Trigger             Trigger                `config:"trigger" validate:"required"`

	entryPoint string
}

// Trigger stores the configuration of the function trigger.
type Trigger struct {
	EventType string `config:"event_type" json:"eventType"`
	Resource  string `config:"resource"  validate:"required" json:"resource"`
	Service   string `config:"service" json:"service,omitempty" yaml:"service,omitempty"`
}

func defaultPubSubFunctionConfig() *FunctionConfig {
	return &FunctionConfig{
		Trigger: Trigger{
			EventType: "google.pubsub.topic.publish",
		},
		entryPoint: "RunPubSub",
	}
}

func defaultStorageFunctionConfig() *FunctionConfig {
	return &FunctionConfig{
		Trigger: Trigger{
			EventType: "google.storage.object.finalize",
		},
		entryPoint: "RunCloudStorage",
	}
}

// Validate checks a function configuration.
func (c *FunctionConfig) Validate() error {
	if c.entryPoint == "" {
		return fmt.Errorf("entryPoint cannot be empty")
	}
	return nil
}

// EntryPoint returns the name of the function to run on GCP.
func (c *FunctionConfig) EntryPoint() string {
	return c.entryPoint
}
