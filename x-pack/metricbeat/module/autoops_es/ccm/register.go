// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ccm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
)

// License information returned from `GET /_license`, contained within the "license" object.
type license struct {
	UID    string `json:"uid"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// License information returned from `GET /_license`
type licenseWrapper struct {
	License license `json:"license"`
}

// Self-Managed Cluster details passed to CCM
type selfManagedClusterInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// License information passed to CCM
type licenseInfo struct {
	UID  string `json:"uid"`
	Type string `json:"type"`
}

type cloudConnectedCluster struct {
	Cluster selfManagedClusterInfo `json:"self_managed_cluster"`
	License licenseInfo            `json:"license"`
}

// Response from CCM after successfully registering a Self-Managed Cluster.
type cloudConnectedResource struct {
	ID string `json:"id"`
}

// The special metricset.getClusterInfo function, which will shutdown the Beat / Agent if
// the version is not supported.
type clusterInfoFetcher func(*elasticsearch.MetricSet) (*utils.ClusterInfo, error)

var (
	SUPPORTED_LICENSE_TYPES = []string{"enterprise", "trial"}
)

// MaybeRegisterCloudConnectedCluster fetches cluster information and license details, then sets the global resource ID if applicable.
func MaybeRegisterCloudConnectedCluster(m *elasticsearch.MetricSet, getClusterInfo clusterInfoFetcher) error {
	// if resource ID is already set, then we don't need to check/register anything
	if utils.GetAndSetResourceID() != "" {
		return nil
	}

	cloudApiKey, err := getCloudConnectedModeApiKey(m)

	if err != nil {
		return fmt.Errorf("failed to get Cloud Connected Mode API key: %w", err)
	} else if cloudApiKey == "" {
		// if there is no Cloud API Key configured, then there is no request to make
		return nil
	}

	m.Logger().Debugf("Attempting to get cluster info for Cloud Connected Mode...")
	clusterInfo, err := getClusterInfo(m)

	if err != nil {
		return fmt.Errorf("failed to load cluster info: %w", err)
	}

	m.Logger().Debugf("Attempting to fetch license for Cloud Connected Mode...")
	licenseWrapper, err := utils.FetchAPIData[licenseWrapper](m, "/_license")

	if err != nil {
		return fmt.Errorf("failed to load cluster license: %w", err)
	} else if licenseWrapper.License.Status != "active" {
		return fmt.Errorf("cluster license is not active: %s", licenseWrapper.License.Status)
	} else if !slices.Contains(SUPPORTED_LICENSE_TYPES, licenseWrapper.License.Type) {
		return fmt.Errorf("cluster license type is not supported: %s", licenseWrapper.License.Type)
	}

	m.Logger().Debugf("Successfully fetched license for Cloud Connected Mode: UUID=%s License=%s", clusterInfo.ClusterID, licenseWrapper.License.UID)
	return registerCloudConnectedCluster(cloudApiKey, clusterInfo, &licenseWrapper.License)
}

// registerCloudConnectedCluster sends the cluster and license information to the CCM API.
// It requires the MetricSet to use its configured HTTP client.
func registerCloudConnectedCluster(cloudApiKey string, clusterInfo *utils.ClusterInfo, license *license) error {
	jsonData, err := json.Marshal(cloudConnectedCluster{
		Cluster: selfManagedClusterInfo{
			ID:      clusterInfo.ClusterID,
			Name:    clusterInfo.ClusterName,
			Version: clusterInfo.Version.Number.String(),
		},
		License: licenseInfo{
			UID:  license.UID,
			Type: license.Type,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to serialize payload for Cloud Connected Mode: %w", err)
	}

	requestURL := getCloudConnectedModeAPIURL() + "/api/v1/cloud-connected/clusters"
	req, err := http.NewRequestWithContext(context.Background(), "POST", requestURL, bytes.NewBuffer(jsonData))

	if err != nil {
		return fmt.Errorf("failed to create HTTP request for Cloud Connected Mode: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "ApiKey "+cloudApiKey)
	req.Header.Set("Content-Type", "application/json")

	data, err := utils.HandleHTTPResponse[cloudConnectedResource](http.DefaultClient.Do(req)) //nolint:bodyclose // the handler closes the body

	if err == nil {
		utils.SetResourceID(data.ID)
		return nil
	}

	return fmt.Errorf("failed to register for Cloud Connected Mode: %w", err)
}
