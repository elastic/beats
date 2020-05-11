// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stackdriver

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes/duration"

	monitoring "cloud.google.com/go/monitoring/apiv3"

	"github.com/pkg/errors"

	"google.golang.org/api/option"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/googlecloud"
)

const (
	// MetricsetName is the name of this Metricset
	MetricsetName = "stackdriver"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet(googlecloud.ModuleName, MetricsetName, New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	config            config
	metricsMeta       map[string]metricMeta
	requester         *stackdriverMetricsRequester
	stackDriverConfig []stackDriverConfig `config:"metrics" validate:"nonzero,required"`
}

//stackDriverConfig holds a configuration specific for stackdriver metricset.
type stackDriverConfig struct {
	MetricTypes []string `config:"metric_types" validate:"required"`
	Aligner     string   `config:"aligner"`
}

type metricMeta struct {
	samplePeriod time.Duration
	ingestDelay  time.Duration
}

type config struct {
	Zone                string `config:"zone"`
	Region              string `config:"region"`
	ProjectID           string `config:"project_id" validate:"required"`
	ExcludeLabels       bool   `config:"exclude_labels"`
	ServiceName         string `config:"stackdriver.service"  validate:"required"`
	CredentialsFilePath string `config:"credentials_file_path"`

	opt    []option.ClientOption
	period duration.Duration
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The gcp '%s' metricset is beta.", MetricsetName)

	m := &MetricSet{BaseMetricSet: base}

	if err := base.Module().UnpackConfig(&m.config); err != nil {
		return nil, err
	}

	stackDriverConfigs := struct {
		StackDriverMetrics []stackDriverConfig `config:"stackdriver.metrics" validate:"nonzero,required"`
	}{}

	if err := base.Module().UnpackConfig(&stackDriverConfigs); err != nil {
		return nil, err
	}

	m.stackDriverConfig = stackDriverConfigs.StackDriverMetrics
	m.config.opt = []option.ClientOption{option.WithCredentialsFile(m.config.CredentialsFilePath)}
	m.config.period.Seconds = int64(m.Module().Config().Period.Seconds())

	if err := validatePeriodForGCP(m.Module().Config().Period); err != nil {
		return nil, err
	}

	// Get ingest delay and sample period for each metric type
	ctx := context.Background()
	client, err := monitoring.NewMetricClient(ctx, m.config.opt...)
	if err != nil {
		return nil, errors.Wrap(err, "error creating Stackdriver client")
	}

	m.metricsMeta, err = m.metricDescriptor(ctx, client)
	if err != nil {
		return nil, errors.Wrap(err, "error calling metricDescriptor function")
	}

	m.requester = &stackdriverMetricsRequester{
		config: m.config,
		client: client,
		logger: m.Logger(),
	}
	return m, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) (err error) {
	responses, err := m.requester.Metrics(ctx, m.stackDriverConfig, m.metricsMeta)
	if err != nil {
		return errors.Wrapf(err, "error trying to get metrics for project '%s' and zone '%s' or region '%s'", m.config.ProjectID, m.config.Zone, m.config.Region)
	}

	events, err := m.eventMapping(ctx, responses)
	if err != nil {
		return err
	}

	for _, event := range events {
		reporter.Event(event)
	}

	return nil
}

func (m *MetricSet) eventMapping(ctx context.Context, tss []timeSeriesWithAligner) ([]mb.Event, error) {
	e := newIncomingFieldExtractor(m.Logger())

	var gcpService = googlecloud.NewStackdriverMetadataServiceForTimeSeries(nil)
	var err error

	if !m.config.ExcludeLabels {
		if gcpService, err = NewMetadataServiceForConfig(m.config); err != nil {
			return nil, errors.Wrap(err, "error trying to create metadata service")
		}
	}

	tsGrouped, err := m.timeSeriesGrouped(ctx, gcpService, tss, e)
	if err != nil {
		return nil, errors.Wrap(err, "error trying to group time series data")
	}

	//Create single events for each group of data that matches some common patterns like labels and timestamp
	events := make([]mb.Event, 0)
	for _, groupedEvents := range tsGrouped {
		event := mb.Event{
			Timestamp:  groupedEvents[0].Timestamp,
			RootFields: groupedEvents[0].ECS,
			ModuleFields: common.MapStr{
				"labels": groupedEvents[0].Labels,
			},
			MetricSetFields: common.MapStr{},
		}

		for _, singleEvent := range groupedEvents {
			event.MetricSetFields.Put(singleEvent.Key, singleEvent.Value)
		}

		events = append(events, event)
	}

	return events, nil
}

// validatePeriodForGCP returns nil if the Period in the module config is in the accepted threshold
func validatePeriodForGCP(d time.Duration) (err error) {
	if d.Seconds() < googlecloud.MonitoringMetricsSamplingRate {
		return errors.Errorf("period in Google Cloud config file cannot be set to less than %d seconds", googlecloud.MonitoringMetricsSamplingRate)
	}

	return nil
}

// Validate stackdriver related config
func (mc *stackDriverConfig) Validate() error {
	gcpAlignerNames := make([]string, 0)
	for k := range googlecloud.AlignersMapToGCP {
		gcpAlignerNames = append(gcpAlignerNames, k)
	}

	if mc.Aligner != "" {
		if _, ok := googlecloud.AlignersMapToGCP[mc.Aligner]; !ok {
			return errors.Errorf("the given aligner is not supported, please specify one of %s as aligner", gcpAlignerNames)
		}
	}
	return nil
}

// metricDescriptor calls ListMetricDescriptorsRequest API to get metric metadata
// (sample period and ingest delay) of each given metric type
func (m *MetricSet) metricDescriptor(ctx context.Context, client *monitoring.MetricClient) (map[string]metricMeta, error) {
	metricsWithMeta := make(map[string]metricMeta, 0)

	for _, sdc := range m.stackDriverConfig {
		for _, mt := range sdc.MetricTypes {
			req := &monitoringpb.ListMetricDescriptorsRequest{
				Name:   "projects/" + m.config.ProjectID,
				Filter: fmt.Sprintf(`metric.type = "%s"`, mt),
			}

			it := client.ListMetricDescriptors(ctx, req)
			out, err := it.Next()
			if err != nil {
				return metricsWithMeta, errors.Errorf("Could not make ListMetricDescriptors request: %s: %v", mt, err)
			}

			// Set samplePeriod default to 60 seconds and ingestDelay default to 0.
			meta := metricMeta{
				samplePeriod: 60 * time.Second,
				ingestDelay:  0 * time.Second,
			}

			if out.Metadata.SamplePeriod != nil {
				m.Logger().Debugf("For metric type %s: sample period = %s", mt, out.Metadata.SamplePeriod)
				meta.samplePeriod = time.Duration(out.Metadata.SamplePeriod.Seconds) * time.Second
			}

			if out.Metadata.IngestDelay != nil {
				m.Logger().Debugf("For metric type %s: ingest delay = %s", mt, out.Metadata.IngestDelay)
				meta.ingestDelay = time.Duration(out.Metadata.IngestDelay.Seconds) * time.Second
			}

			metricsWithMeta[mt] = meta
		}
	}

	return metricsWithMeta, nil
}
