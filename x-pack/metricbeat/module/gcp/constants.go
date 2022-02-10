// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"

const (
	// ModuleName in Metricbeat
	ModuleName = "gcp"

	// MonitoringMetricsSamplingRate (in second) refers to how frequent monitoring collects measurement in GCP.
	MonitoringMetricsSamplingRate = 60
)

// Metricsets / GCP services names
const (
	ServiceCloudFunctions = "cloudfunctions"
	ServiceCompute        = "compute"
	ServiceGKE            = "gke"
	ServiceLoadBalancing  = "loadbalancing"
	ServicePubsub         = "pubsub"
	ServiceStorage        = "storage"
<<<<<<< HEAD
=======
	ServiceFirestore      = "firestore"
	ServiceDataproc       = "dataproc"
>>>>>>> e61173f979 ([Metricbeat] gcp: add dataproc metricset (#30008))
)

//Paths within the GCP monitoring.TimeSeries response, if converted to JSON, where you can find each ECS field required for the output event
const (
	TimeSeriesResponsePathForECSAvailabilityZone = "zone"
	TimeSeriesResponsePathForECSAccountID        = "project_id"
	TimeSeriesResponsePathForECSInstanceID       = "instance_id"
	TimeSeriesResponsePathForECSInstanceName     = "instance_name"
)

// ECS Fields that are being captured from responses
const (
	//Cloud fields https://www.elastic.co/guide/en/ecs/master/ecs-cloud.html
	ECSCloud = "cloud"

	ECSCloudAvailabilityZone = "availability_zone"

	ECSCloudProvider = "provider"

	ECSCloudRegion = "region"

	ECSCloudAccount     = "account"
	ECSCloudAccountID   = "id"
	ECSCloudAccountName = "name"

	ECSCloudInstance        = "instance"
	ECSCloudInstanceKey     = ECSCloud + "." + ECSCloudInstance
	ECSCloudInstanceID      = "id"
	ECSCloudInstanceIDKey   = ECSCloudInstanceKey + "." + ECSCloudInstanceID
	ECSCloudInstanceName    = "name"
	ECSCloudInstanceNameKey = ECSCloudInstanceKey + "." + ECSCloudInstanceName

	ECSCloudMachine        = "machine"
	ECSCloudMachineKey     = ECSCloud + "." + ECSCloudMachine
	ECSCloudMachineType    = "type"
	ECSCloudMachineTypeKey = ECSCloudMachineKey + "." + ECSCloudMachineType
)

// Metadata keys used for events. They follow GCP structure.
const (
	//Stackdriver
	LabelMetrics      = "metrics"
	LabelResource     = "resource"
	LabelSystem       = "system"
	LabelUserMetadata = "metadata.user"
	KeyTimestamp      = "timestamp"

	// Compute
	LabelUser     = "user"
	LabelMetadata = "metadata"
)

// Available perSeriesAligner map
// https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.alertPolicies#Aligner
var AlignersMapToGCP = map[string]monitoringpb.Aggregation_Aligner{
	"ALIGN_NONE":           monitoringpb.Aggregation_ALIGN_NONE,
	"ALIGN_DELTA":          monitoringpb.Aggregation_ALIGN_DELTA,
	"ALIGN_RATE":           monitoringpb.Aggregation_ALIGN_RATE,
	"ALIGN_INTERPOLATE":    monitoringpb.Aggregation_ALIGN_INTERPOLATE,
	"ALIGN_NEXT_OLDER":     monitoringpb.Aggregation_ALIGN_NEXT_OLDER,
	"ALIGN_MIN":            monitoringpb.Aggregation_ALIGN_MIN,
	"ALIGN_MAX":            monitoringpb.Aggregation_ALIGN_MAX,
	"ALIGN_MEAN":           monitoringpb.Aggregation_ALIGN_MEAN,
	"ALIGN_COUNT":          monitoringpb.Aggregation_ALIGN_COUNT,
	"ALIGN_SUM":            monitoringpb.Aggregation_ALIGN_SUM,
	"ALIGN_STDDEV":         monitoringpb.Aggregation_ALIGN_STDDEV,
	"ALIGN_COUNT_TRUE":     monitoringpb.Aggregation_ALIGN_COUNT_TRUE,
	"ALIGN_COUNT_FALSE":    monitoringpb.Aggregation_ALIGN_COUNT_FALSE,
	"ALIGN_FRACTION_TRUE":  monitoringpb.Aggregation_ALIGN_FRACTION_TRUE,
	"ALIGN_PERCENTILE_99":  monitoringpb.Aggregation_ALIGN_PERCENTILE_99,
	"ALIGN_PERCENTILE_95":  monitoringpb.Aggregation_ALIGN_PERCENTILE_95,
	"ALIGN_PERCENTILE_50":  monitoringpb.Aggregation_ALIGN_PERCENTILE_50,
	"ALIGN_PERCENTILE_05":  monitoringpb.Aggregation_ALIGN_PERCENTILE_05,
	"ALIGN_PERCENT_CHANGE": monitoringpb.Aggregation_ALIGN_PERCENT_CHANGE,
}

const (
	DefaultAligner = "ALIGN_NONE"
)

var AlignersMapToSuffix = map[string]string{
	"ALIGN_NONE":           ".value",
	"ALIGN_DELTA":          ".delta",
	"ALIGN_RATE":           ".rate",
	"ALIGN_INTERPOLATE":    ".interpolate",
	"ALIGN_NEXT_OLDER":     ".next_older",
	"ALIGN_MIN":            ".min",
	"ALIGN_MAX":            ".max",
	"ALIGN_MEAN":           ".avg",
	"ALIGN_COUNT":          ".count",
	"ALIGN_SUM":            ".sum",
	"ALIGN_STDDEV":         ".stddev",
	"ALIGN_COUNT_TRUE":     ".count_true",
	"ALIGN_COUNT_FALSE":    ".count_false",
	"ALIGN_FRACTION_TRUE":  ".fraction_true",
	"ALIGN_PERCENTILE_99":  ".percentile_99",
	"ALIGN_PERCENTILE_95":  ".percentile_95",
	"ALIGN_PERCENTILE_50":  ".percentile_50",
	"ALIGN_PERCENTILE_05":  ".percentile_05",
	"ALIGN_PERCENT_CHANGE": ".percent_change",
}
