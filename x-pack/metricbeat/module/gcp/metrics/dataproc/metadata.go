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

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp"
	"github.com/elastic/elastic-agent-libs/logp"
)

// NewMetadataService returns the specific Metadata service for a GCP Dataproc cluster
func NewMetadataService(projectID string, regions []string, organizationID, organizationName string, projectName string, collectUserLabels bool, opt ...option.ClientOption) (gcp.MetadataService, error) {
	return &metadataCollector{
		projectID:         projectID,
		projectName:       projectName,
		organizationID:    organizationID,
		organizationName:  organizationName,
		regions:           regions,
		collectUserLabels: collectUserLabels,
		opt:               opt,
		clusters:          make(map[string]*dataproc.Cluster),
		logger:            logp.NewLogger("metrics-dataproc"),
	}, nil
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
	// NOTE: clusters holds data used for all metrics collected in a given period
	// this avoids calling the remote endpoint for each metric, which would take a long time overall
	clusters map[string]*dataproc.Cluster
	logger   *logp.Logger
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
	cluster, err := s.instance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("error trying to get data from instance '%s' %w", instanceID, err)
	}

	metadata := &dataprocMetadata{
		clusterID: instanceID,
		region:    region,
	}

	if cluster == nil {
		s.logger.Debugf("couldn't get instance '%s' call ListInstances API", instanceID)
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

// instance returns data from an instance ID from the cache.
func (s *metadataCollector) instance(ctx context.Context, instanceID string) (*dataproc.Cluster, error) {
	s.getInstances(ctx)

	instance, ok := s.clusters[instanceID]
	if ok {
		return instance, nil
	}

	s.clusters = make(map[string]*dataproc.Cluster)

	return nil, nil
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

func (s *metadataCollector) getInstances(ctx context.Context) {
	if len(s.clusters) > 0 {
		return
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	regionsToQuery := s.regions
	if len(regionsToQuery) == 0 {
		regions, err := s.fetchAvailableRegions(ctx)
		if err != nil {
			s.logger.Errorf("error fetching available regions: %v", err)
			return
		}
		regionsToQuery = regions
	}

	dataprocService, err := dataproc.NewService(ctx, s.opt...)
	if err != nil {
		s.logger.Errorf("error creating dataproc service %v", err)
		return
	}
	clustersService := dataproc.NewProjectsRegionsClustersService(dataprocService)

	s.logger.Debugf("querying dataproc clusters across %d regions: %v", len(regionsToQuery), regionsToQuery)

	queryCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	for _, region := range regionsToQuery {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()

			listCall := clustersService.List(s.projectID, region).Fields("clusters.labels", "clusters.clusterUuid").Context(queryCtx)
			resp, err := listCall.Do()
			if err != nil {
				s.logger.Errorf("dataproc ListClusters error in region %s: %v", region, err)
				return
			}

			for _, cluster := range resp.Clusters {
				mu.Lock()
				s.clusters[cluster.ClusterUuid] = cluster
				mu.Unlock()
			}
		}(region)
	}

	wg.Wait()
	s.logger.Debugf("completed fetching dataproc clusters, found %d clusters", len(s.clusters))
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
