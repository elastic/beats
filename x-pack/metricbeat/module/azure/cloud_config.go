// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package azure

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
)

const (
	// GovCloudBaseURI is the resource manager endpoint for the Azure Government cloud.
	GovCloudBaseURI = "https://management.usgovcloudapi.net/"
	// ChinaCloudBaseURI is the resource manager endpoint for the Azure China cloud.
	ChinaCloudBaseURI = "https://management.chinacloudapi.cn/"
)

// baseCloud maps a resource manager endpoint to one of the SDK's predefined
// cloud configurations, so that every Azure service (ARM, Monitor metrics
// batch API, ...) gets a consistent endpoint and token audience. The second
// return value reports whether the endpoint matched a known cloud.
func baseCloud(resourceManagerEndpoint string) (cloud.Configuration, bool) {
	switch resourceManagerEndpoint {
	case "", DefaultBaseURI:
		return cloud.AzurePublic, true
	case GovCloudBaseURI:
		return cloud.AzureGovernment, true
	case ChinaCloudBaseURI:
		return cloud.AzureChina, true
	default:
		return cloud.AzurePublic, false
	}
}

// BuildCloudConfig builds the cloud configuration for the SDK clients from
// the module config. The base cloud is selected from resource_manager_endpoint,
// and resource_manager_endpoint/resource_manager_audience act as overrides on
// top of it for non-standard environments (e.g. Azure Stack).
func BuildCloudConfig(config Config) cloud.Configuration {
	base, _ := baseCloud(config.ResourceManagerEndpoint)

	// Deep-copy the services map: the SDK's predefined configurations are
	// package-level globals shared by the whole process and must not be mutated.
	services := make(map[cloud.ServiceName]cloud.ServiceConfiguration, len(base.Services))
	for name, svc := range base.Services {
		services[name] = svc
	}

	resourceManager := services[cloud.ResourceManager]
	if config.ResourceManagerEndpoint != "" && config.ResourceManagerEndpoint != DefaultBaseURI {
		resourceManager.Endpoint = config.ResourceManagerEndpoint
	}
	if config.ResourceManagerAudience != "" {
		resourceManager.Audience = config.ResourceManagerAudience
	}
	services[cloud.ResourceManager] = resourceManager

	return cloud.Configuration{
		ActiveDirectoryAuthorityHost: config.ActiveDirectoryEndpoint,
		Services:                     services,
	}
}

// metricsBatchEndpoint returns the regional Azure Monitor metrics batch API
// endpoint for the cloud identified by the resource manager endpoint. It
// returns an error for unknown clouds instead of silently defaulting to the
// public cloud endpoint.
func metricsBatchEndpoint(resourceManagerEndpoint string, location string) (string, error) {
	base, known := baseCloud(resourceManagerEndpoint)
	if !known {
		return "", fmt.Errorf("the metrics batch API is not supported for the resource manager endpoint %q: no known metrics endpoint for this cloud", resourceManagerEndpoint)
	}

	var suffix string
	switch base.ActiveDirectoryAuthorityHost {
	case cloud.AzureGovernment.ActiveDirectoryAuthorityHost:
		suffix = "azure.us"
	case cloud.AzureChina.ActiveDirectoryAuthorityHost:
		suffix = "azure.cn"
	default:
		suffix = "azure.com"
	}

	return fmt.Sprintf("https://%s.metrics.monitor.%s", location, suffix), nil
}
