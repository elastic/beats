// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package dataproc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/api/dataproc/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/elastic/beats/v9/libbeat/common/backoff"
	"github.com/elastic/beats/v9/x-pack/metricbeat/module/gcp"
	"github.com/elastic/elastic-agent-libs/logp"
)

// NewMetadataService returns the specific Metadata service for a GCP Dataproc cluster
func NewMetadataService(
	ctx context.Context,
	projectID string,
	regions []string,
	organizationID, organizationName, projectName string,
	collectUserLabels bool,
	cacheRegistry *gcp.CacheRegistry,
	logger *logp.Logger,
	opt ...option.ClientOption) (gcp.MetadataService, error) {
	mc := &metadataCollector{
		projectID:         projectID,
		projectName:       projectName,
		organizationID:    organizationID,
		organizationName:  organizationName,
		regions:           regions,
		collectUserLabels: collectUserLabels,
		opt:               opt,
		instanceCache:     cacheRegistry.Dataproc,
		logger:            logger.Named("metrics-dataproc"),
	}

	// Freshen up the cache, later all we have to do is look up the instance
	err := mc.instanceCache.EnsureFresh(func() (map[string]*dataproc.Cluster, error) {
		instances := make(map[string]*dataproc.Cluster)
		r := backoff.NewRetryer(3, time.Second, 30*time.Second)

		err := r.Retry(ctx, func() error {
			var err error
			instances, err = mc.fetchDataprocClusters(ctx)
			return err
		})

		return instances, err
	})

	return mc, err
}

// dataprocMetadata is an object to store data in between the extraction and the writing in the destination (to uncouple
// reading and writing in the same method)
type dataprocMetadata struct {
	region      string
	clusterID   string
	clusterName string
	machineType string

	User     map[string]string
	Metadata map[string]string
	Metrics  interface{}
	System   interface{}
}

type metadataCollector struct {
	projectID         string
	projectName       string
	organizationID    string
	organizationName  string
	regions           []string
	collectUserLabels bool
	opt               []option.ClientOption
	instanceCache     *gcp.Cache[*dataproc.Cluster]
	logger            *logp.Logger
}

// Metadata implements googlecloud.MetadataCollector to the known set of labels from a Dataproc TimeSeries single point of data.
func (s *metadataCollector) Metadata(ctx context.Context, resp *monitoringpb.TimeSeries) (gcp.MetadataCollectorData, error) {
	stackdriverLabels := gcp.NewStackdriverMetadataServiceForTimeSeries(resp, s.organizationID, s.organizationName, s.projectName)

	metadataCollectorData, err := stackdriverLabels.Metadata(ctx, resp)
	if err != nil {
		return gcp.MetadataCollectorData{}, err
	}

	if resp.Resource != nil && resp.Resource.Labels != nil {
		_, _ = metadataCollectorData.ECS.Put(gcp.ECSCloudInstanceIDKey, resp.Resource.Labels["cluster_uuid"])
	}

	_, _ = metadataCollectorData.ECS.Put(gcp.ECSCloudInstanceNameKey, resp.Resource.Labels["cluster_name"])

	if s.collectUserLabels {
		metadata, err := s.instanceMetadata(ctx, s.instanceID(resp), s.instanceRegion(resp))
		if err != nil {
			return metadataCollectorData, err
		}

		if metadata.machineType != "" {
			lastIndex := strings.LastIndex(metadata.machineType, "/")
			_, _ = metadataCollectorData.ECS.Put(gcp.ECSCloudMachineTypeKey, metadata.machineType[lastIndex+1:])
		}

		metadata.Metrics = metadataCollectorData.Labels[gcp.LabelMetrics]
		metadata.System = metadataCollectorData.Labels[gcp.LabelSystem]

		if metadata.User != nil {
			metadataCollectorData.Labels[gcp.LabelUser] = metadata.User
		}
	}

	return metadataCollectorData, nil
}

// instanceMetadata returns the labels of an instance
func (s *metadataCollector) instanceMetadata(ctx context.Context, instanceID, region string) (*dataprocMetadata, error) {
	metadata := &dataprocMetadata{
		clusterID: instanceID,
		region:    region,
	}

	cluster, ok := s.instanceCache.Get(instanceID)
	if !ok {
		s.logger.Warnf("Cluster %s not found in dataproc cache.", instanceID)
		return metadata, nil
	}

	if cluster.ClusterName != "" {
		parts := strings.Split(cluster.ClusterName, "/")
		metadata.clusterName = parts[len(parts)-1]
	}

	if cluster.Labels != nil {
		metadata.User = cluster.Labels
	}

	return metadata, nil
}

func (s *metadataCollector) instanceID(ts *monitoringpb.TimeSeries) string {
	if ts.Resource != nil && ts.Resource.Labels != nil {
		return ts.Resource.Labels["cluster_uuid"]
	}

	return ""
}

func (s *metadataCollector) instanceRegion(ts *monitoringpb.TimeSeries) string {
	if ts.Resource != nil && ts.Resource.Labels != nil {
		return ts.Resource.Labels["region"]
	}

	return ""
}

func (s *metadataCollector) fetchDataprocClusters(ctx context.Context) (map[string]*dataproc.Cluster, error) {
	fetchedClusters := make(map[string]*dataproc.Cluster)

	var wg sync.WaitGroup
	var mu sync.Mutex

	regionsToQuery := s.regions
	if len(regionsToQuery) == 0 {
		regions, err := s.fetchAvailableRegions(ctx)
		if err != nil {
			s.logger.Errorf("error fetching available regions: %v", err)
			return nil, fmt.Errorf("error fetching available regions for Dataproc: %w", err)
		}
		regionsToQuery = regions
	}

	dataprocService, err := dataproc.NewService(ctx, s.opt...)
	if err != nil {
		s.logger.Errorf("error creating dataproc service %v", err)
		return nil, fmt.Errorf("error creating dataproc service: %w", err)
	}
	clustersService := dataproc.NewProjectsRegionsClustersService(dataprocService)

	s.logger.Debugf("querying dataproc clusters across %d regions: %v", len(regionsToQuery), regionsToQuery)

	queryCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	for _, region := range regionsToQuery {
		wg.Add(1)
		go func(currentRegion string) {
			defer wg.Done()

			listCall := clustersService.List(s.projectID, currentRegion).Fields("clusters.labels", "clusters.clusterUuid").Context(queryCtx)
			resp, err := listCall.Do()
			if err != nil {
				s.logger.Errorf("dataproc ListClusters error in region %s: %v", currentRegion, err)
				return
			}

			mu.Lock()
			for _, cluster := range resp.Clusters {
				fetchedClusters[cluster.ClusterUuid] = cluster
			}
			mu.Unlock()
		}(region)
	}

	wg.Wait()
	s.logger.Debugf("completed fetching dataproc clusters, found %d clusters", len(fetchedClusters))
	return fetchedClusters, nil
}

// fetchAvailableRegions gets all available GCP regions
func (s *metadataCollector) fetchAvailableRegions(ctx context.Context) ([]string, error) {
	restClient, err := compute.NewRegionsRESTClient(ctx, s.opt...)
	if err != nil {
		return nil, fmt.Errorf("error getting client from compute regions service: %w", err)
	}
	defer restClient.Close()

	regionsIt := restClient.List(ctx, &computepb.ListRegionsRequest{
		Project: s.projectID,
	})

	var regions []string
	for {
		region, err := regionsIt.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error getting next region from regions iterator: %w", err)
		}

		// Only include regions that are UP
		if region.GetStatus() == "UP" {
			regions = append(regions, region.GetName())
		}
	}

	s.logger.Debugf("found %d available regions: %v", len(regions), regions)
	return regions, nil
}
