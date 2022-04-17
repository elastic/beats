// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"time"

	"github.com/menderesk/beats/v7/libbeat/common"

	"github.com/pkg/errors"
)

const (
	// DefaultBaseURI is the default URI used for the service Insights
	DefaultBaseURI = "https://management.azure.com/"
)

var (
	AzureEnvs = common.MapStr{
		"https://management.azure.com/":         "https://login.microsoftonline.com/",
		"https://management.usgovcloudapi.net/": "https://login.microsoftonline.us/",
		"https://management.chinacloudapi.cn/":  "https://login.chinacloudapi.cn/",
		"https://management.microsoftazure.de/": "https://login.microsoftonline.de/",
	}
)

// Config options
type Config struct {
	// shared config options
	ClientId                string        `config:"client_id"  validate:"required"`
	ClientSecret            string        `config:"client_secret"  validate:"required"`
	TenantId                string        `config:"tenant_id"  validate:"required"`
	SubscriptionId          string        `config:"subscription_id"  validate:"required"`
	Period                  time.Duration `config:"period" validate:"nonzero,required"`
	ResourceManagerEndpoint string        `config:"resource_manager_endpoint"`
	ActiveDirectoryEndpoint string        `config:"active_directory_endpoint"`
	// specific to resource metrics
	Resources           []ResourceConfig `config:"resources"`
	RefreshListInterval time.Duration    `config:"refresh_list_interval"`
	DefaultResourceType string           `config:"default_resource_type"`
	AddCloudMetadata    bool             `config:"add_cloud_metadata"`
	// specific to billing
	BillingScopeDepartment string `config:"billing_scope_department"` // retrieve usage details from department scope
	BillingScopeAccountId  string `config:"billing_scope_account_id"` // retrieve usage details from billing account ID scope
}

// ResourceConfig contains resource and metric list specific configuration.
type ResourceConfig struct {
	Id          []string       `config:"resource_id"`
	Group       []string       `config:"resource_group"`
	Metrics     []MetricConfig `config:"metrics"`
	Type        string         `config:"resource_type"`
	Query       string         `config:"resource_query"`
	ServiceType []string       `config:"service_type"`
}

// MetricConfig contains metric specific configuration.
type MetricConfig struct {
	Name         []string          `config:"name"`
	Namespace    string            `config:"namespace"`
	Aggregations []string          `config:"aggregations"`
	Dimensions   []DimensionConfig `config:"dimensions"`
	Timegrain    string            `config:"timegrain"`
	// namespaces can be unsupported by some resources and supported in some, this configuration option makes sure no error messages are returned if namespace is unsupported
	// info messages will be logged instead. Same situation with metrics, some are being removed from the API, we would like to make sure that does not affect the module
	IgnoreUnsupported bool `config:"ignore_unsupported"`
}

// DimensionConfig contains dimensions specific configuration.
type DimensionConfig struct {
	Name  string `config:"name"`
	Value string `config:"value"`
}

func (conf *Config) Validate() error {
	if conf.ResourceManagerEndpoint == "" {
		conf.ResourceManagerEndpoint = DefaultBaseURI
	}
	if conf.ActiveDirectoryEndpoint == "" {
		ok, err := AzureEnvs.HasKey(conf.ResourceManagerEndpoint)
		if err != nil {
			return errors.Wrap(err, "No active directory endpoint found for the resource manager endpoint selected.")
		}
		if ok {
			add, err := AzureEnvs.GetValue(conf.ResourceManagerEndpoint)
			if err != nil {
				return errors.Wrap(err, "No active directory endpoint found for the resource manager endpoint selected.")
			}
			conf.ActiveDirectoryEndpoint = add.(string)
		}
		if conf.ActiveDirectoryEndpoint == "" {
			return errors.New("no active directory endpoint has been configured")
		}
	}
	return nil
}
