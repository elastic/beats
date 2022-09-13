package cloudsql

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"google.golang.org/api/option"
	"google.golang.org/api/sqladmin/v1"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

const (
	cacheTTL         = 30 * time.Second
	initialCacheSize = 13
)

// NewMetadataService returns the specific Metadata service for a GCP CloudSQL resource.
func NewMetadataService(projectID, zone string, region string, regions []string, opt ...option.ClientOption) (gcp.MetadataService, error) {
	return &metadataCollector{
		projectID:     projectID,
		zone:          zone,
		region:        region,
		regions:       regions,
		opt:           opt,
		instanceCache: common.NewCache(cacheTTL, initialCacheSize),
		logger:        logp.NewLogger("metrics-cloudsql"),
	}, nil
}

// cloudsqlMetadata is an object to store data in between the extraction and the writing in the destination (to uncouple
// reading and writing in the same method)
type cloudsqlMetadata struct {
	// projectID   string
	region          string
	instanceID      string
	machineType     string
	databaseVersion string

	// ts *monitoringpb.TimeSeries

	User     map[string]string
	Metadata map[string]string
	Metrics  interface{}
	System   interface{}
}

type metadataCollector struct {
	projectID     string
	zone          string
	region        string
	regions       []string
	opt           []option.ClientOption
	instanceCache *common.Cache
	logger        *logp.Logger
}

func getDatabaseNameAndVersion(db string) mapstr.M {
	parts := strings.SplitN(strings.ToLower(db), "_", 2)

	var cloudsqlDb mapstr.M

	switch {
	case db == "SQL_DATABASE_VERSION_UNSPECIFIED":
		cloudsqlDb = mapstr.M{
			"name":    "sql",
			"version": "unspecified",
		}
	case strings.Contains(parts[0], "sqlserver"):
		cloudsqlDb = mapstr.M{
			"name":    strings.ToLower(parts[0]),
			"version": strings.ToLower(parts[1]),
		}
	default:
		version := strings.ReplaceAll(parts[1], "_", ".")
		cloudsqlDb = mapstr.M{
			"name":    strings.ToLower(parts[0]),
			"version": version,
		}
	}

	return cloudsqlDb
}

// Metadata implements googlecloud.MetadataCollector to the known set of labels from a CloudSQL TimeSeries single point of data.
func (s *metadataCollector) Metadata(ctx context.Context, resp *monitoringpb.TimeSeries) (gcp.MetadataCollectorData, error) {
	cloudsqlMetadata, err := s.instanceMetadata(ctx, s.instanceID(resp), s.instanceRegion(resp))
	if err != nil {
		return gcp.MetadataCollectorData{}, err
	}

	stackdriverLabels := gcp.NewStackdriverMetadataServiceForTimeSeries(resp)

	metadataCollectorData, err := stackdriverLabels.Metadata(ctx, resp)
	if err != nil {
		return gcp.MetadataCollectorData{}, err
	}

	if cloudsqlMetadata.machineType != "" {
		lastIndex := strings.LastIndex(cloudsqlMetadata.machineType, "/")
		_, _ = metadataCollectorData.ECS.Put(gcp.ECSCloudMachineTypeKey, cloudsqlMetadata.machineType[lastIndex+1:])
	}

	cloudsqlMetadata.Metrics = metadataCollectorData.Labels[gcp.LabelMetrics]
	cloudsqlMetadata.System = metadataCollectorData.Labels[gcp.LabelSystem]

	if cloudsqlMetadata.databaseVersion != "" {
		err := mapstr.MergeFields(metadataCollectorData.Labels, mapstr.M{
			"cloudsql": getDatabaseNameAndVersion(cloudsqlMetadata.databaseVersion),
		}, true)
		if err != nil {
			s.logger.Warnf("failed merging cloudsql to label fields: %w", err)
		}
	}

	return metadataCollectorData, nil
}

func (s *metadataCollector) instanceID(ts *monitoringpb.TimeSeries) string {
	if ts.Resource != nil && ts.Resource.Labels != nil {
		return ts.Resource.Labels["database_id"]
	}

	return ""
}

func (s *metadataCollector) instanceRegion(ts *monitoringpb.TimeSeries) string {
	if ts.Resource != nil && ts.Resource.Labels != nil {
		return ts.Resource.Labels["region"]
	}

	return ""
}

// instanceMetadata returns the labels of an instance
func (s *metadataCollector) instanceMetadata(ctx context.Context, instanceID, region string) (*cloudsqlMetadata, error) {
	instance, err := s.instance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("error trying to get data from instance '%s' %w", instanceID, err)
	}

	cloudsqlMetadata := &cloudsqlMetadata{
		instanceID: instanceID,
		region:     region,
	}

	if instance == nil {
		return cloudsqlMetadata, nil
	}

	if instance.DatabaseVersion != "" {
		cloudsqlMetadata.databaseVersion = instance.DatabaseVersion
	}

	return cloudsqlMetadata, nil
}

func (s *metadataCollector) refreshInstanceCache(ctx context.Context) {
	// only refresh cache if it is empty
	if s.instanceCache.Size() > 0 {
		return
	}

	s.logger.Debugf("refresh cache with Instances.List API")

	service, _ := sqladmin.NewService(ctx, s.opt...)

	req := service.Instances.List(s.projectID)
	if err := req.Pages(ctx, func(page *sqladmin.InstancesListResponse) error {
		for _, instancesScopedList := range page.Items {
			s.instanceCache.Put(fmt.Sprintf("%s:%s", instancesScopedList.Project, instancesScopedList.Name), instancesScopedList)
		}
		return nil
	}); err != nil {
		s.logger.Errorf("cloudsql Instances.List error: %v", err)
	}
}

func (s *metadataCollector) instance(ctx context.Context, instanceName string) (*sqladmin.DatabaseInstance, error) {
	s.refreshInstanceCache(ctx)
	instanceCachedData := s.instanceCache.Get(instanceName)
	if instanceCachedData != nil {
		if cloudsqlInstance, ok := instanceCachedData.(*sqladmin.DatabaseInstance); ok {
			return cloudsqlInstance, nil
		}
	}

	return nil, nil
}

func (s *metadataCollector) ID(ctx context.Context, in *gcp.MetadataCollectorInputData) (string, error) {
	metadata, err := s.Metadata(ctx, in.TimeSeries)
	if err != nil {
		return "", err
	}

	metadata.ECS.Update(metadata.Labels)
	if in.Timestamp != nil {
		metadata.ECS.Put("timestamp", in.Timestamp)
	} else if in.Point != nil {
		metadata.ECS.Put("timestamp", in.Point.Interval.EndTime)
	} else {
		return "", errors.New("no timestamp information found")
	}

	return metadata.ECS.String(), nil
}
