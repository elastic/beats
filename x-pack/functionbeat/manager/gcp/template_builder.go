// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	cloudfunctions "google.golang.org/api/cloudfunctions/v1"
	yaml "gopkg.in/yaml.v2"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/functionbeat/function/provider"
	"github.com/elastic/beats/x-pack/functionbeat/manager/core"
	"github.com/elastic/beats/x-pack/functionbeat/manager/core/bundle"
	fngcp "github.com/elastic/beats/x-pack/functionbeat/provider/gcp/gcp"
)

const (
	runtime          = "go111"                            // Golang 1.11
	archiveURL       = "gs://%s/%s"                       // path to the function archive
	locationTemplate = "projects/%s/locations/%s"         // full name of the location
	functionName     = locationTemplate + "/functions/%s" // full name of the functions

	// Package size limits for GCP provider
	packageCompressedLimit   = 100 * 1000 * 1000 // 100MB
	packageUncompressedLimit = 500 * 1000 * 1000 // 500MB
)

// defaultTemplateBuilder builds request object when deploying Functionbeat using
// the command deploy.
type defaultTemplateBuilder struct {
	provider  provider.Provider
	log       *logp.Logger
	gcpConfig *Config
}

type functionData struct {
	raw      []byte
	function *cloudfunctions.CloudFunction
}

// NewTemplateBuilder returns the requested template builder
func NewTemplateBuilder(log *logp.Logger, cfg *common.Config, p provider.Provider) (provider.TemplateBuilder, error) {
	gcpCfg := &Config{}
	err := cfg.Unpack(gcpCfg)
	if err != nil {
		return &defaultTemplateBuilder{}, err
	}

	return &defaultTemplateBuilder{log: log, gcpConfig: gcpCfg, provider: p}, nil
}

func (d *defaultTemplateBuilder) execute(name string) (*functionData, error) {
	d.log.Debug("Compressing all assets into an artifact")

	fn, err := findFunction(d.provider, name)
	if err != nil {
		return nil, err
	}

	resources := zipResourcesOfFunc(fn.Name())
	raw, err := core.MakeZip(packageUncompressedLimit, packageCompressedLimit, resources)
	if err != nil {
		return nil, err
	}

	d.log.Debugf("Compression is successful (zip size: %d bytes)", len(raw))

	return &functionData{
		raw:      raw,
		function: d.cloudFunction(name, fn.Config()),
	}, nil
}

func findFunction(p provider.Provider, name string) (installer, error) {
	fn, err := p.FindFunctionByName(name)
	if err != nil {
		return nil, err
	}

	function, ok := fn.(installer)
	if !ok {
		return nil, errors.New("incompatible type received, expecting: 'functionManager'")
	}

	return function, nil
}

func (d *defaultTemplateBuilder) cloudFunction(name string, config *fngcp.FunctionConfig) *cloudfunctions.CloudFunction {
	fnName := fmt.Sprintf(functionName, d.gcpConfig.ProjectID, d.gcpConfig.Location, name)
	sourceArchiveURL := fmt.Sprintf(archiveURL, d.gcpConfig.FunctionStorage, name)

	return &cloudfunctions.CloudFunction{
		Name:        fnName,
		Description: config.Description,
		EntryPoint:  config.EntryPoint(),
		EnvironmentVariables: map[string]string{
			"ENABLED_FUNCTIONS": name,
			"BEAT_STRICT_PERMS": "false",
		},
		EventTrigger: &cloudfunctions.EventTrigger{
			EventType: config.Trigger.EventType,
			Resource:  config.Trigger.Resource,
			Service:   config.Trigger.Service,
		},
		Labels:              config.Labels,
		MaxInstances:        int64(config.MaxInstances),
		Runtime:             runtime,
		ServiceAccountEmail: config.ServiceAccountEmail,
		SourceArchiveUrl:    sourceArchiveURL,
		Timeout:             config.Timeout.String(),
		VpcConnector:        config.VPCConnector,
	}
}

// RawTemplate returns the JSON to POST to the endpoint.
func (d *defaultTemplateBuilder) RawTemplate(name string) (string, error) {
	fn, err := findFunction(d.provider, name)
	if err != nil {
		return "", err
	}
	config := fn.Config()

	properties := common.MapStr{
		"codeLocation":     "pkg/" + fn.Name(),
		"codeBucket":       d.gcpConfig.FunctionStorage,
		"codeBucketObject": "functionbeat.zip",
		"location":         d.gcpConfig.Location,
		"runtime":          runtime,
		"entryPoint":       config.EntryPoint(),
		"eventTrigger":     config.Trigger,
		"environmentVariables": common.MapStr{
			"ENABLED_FUNCTIONS": name,
			"BEAT_STRICT_PERMS": false,
		},
	}

	if config.Timeout > 0*time.Second {
		properties["timeout"] = config.Timeout.String()
	}
	if config.MemorySize != "" {
		properties["availableMemoryMb"] = config.MemorySize
	}
	if len(config.ServiceAccountEmail) > 0 {
		properties["serviceAccountEmail"] = config.ServiceAccountEmail
	}
	if len(config.Labels) > 0 {
		properties["labels"] = config.Labels
	}
	if config.MaxInstances > 0 {
		properties["maxInstances"] = config.MaxInstances
	}
	if len(config.VPCConnector) > 0 {
		properties["vpcConnector"] = config.VPCConnector
	}

	output := common.MapStr{
		"resources": []common.MapStr{
			common.MapStr{
				"name":       fmt.Sprintf(functionName, d.gcpConfig.ProjectID, d.gcpConfig.Location, name),
				"type":       "cloudfunctions.v1beta2.function", // TODO
				"properties": properties,
			},
		},
	}

	yamlBytes, err := yaml.Marshal(output)
	return string(yamlBytes), err
}

func zipResources() map[string][]bundle.Resource {
	functions, err := provider.ListFunctions("gcp")
	if err != nil {
		fmt.Println(err)
		return nil
	}

	resources := make(map[string][]bundle.Resource)
	for _, f := range functions {
		resources["gcp-"+f] = zipResourcesOfFunc(f)
	}
	return resources
}

func zipResourcesOfFunc(typeName string) []bundle.Resource {
	root := filepath.Join("pkg", typeName)
	vendor := bundle.Folder(filepath.Join("pkg", typeName, "vendor"), filepath.Join("pkg", typeName), 0644)
	return append(vendor, &bundle.LocalFile{Path: filepath.Join(root, typeName+".go"), FileMode: 0755})
}
