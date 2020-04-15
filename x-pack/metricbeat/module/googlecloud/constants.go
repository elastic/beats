// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package googlecloud

const (
	// ModuleName in Metricbeat
	ModuleName = "googlecloud"

	// MinTimeIntervalDataWindowMinutes is the minimum time in minutes that we allow the user to specify when requesting past metrics. Less than 5 minutes
	// usually return no results.
	MinTimeIntervalDataWindowMinutes = 1

	// MaxTimeIntervalDataWindowMinutes is the max time in minutes that we allow the user to specify when requesting past metrics.
	MaxTimeIntervalDataWindowMinutes = 60

	// MonitoringMetricsLatency (in minute) refers to how long it takes before a new metric data point is available in Monitoring after it is written.
	// Monitoring collects one measurement each minute (the sampling rate), but it can take up to 4 minutes before you can retrieve the data (latency).
	// So, the time stamp recording the collection time might be up to 4 minutes old.
	MonitoringMetricsLatency = 4

	// MonitoringMetricsSamplingRate (in second) refers to how frequent monitoring collects measurement in GCP.
	MonitoringMetricsSamplingRate = 60
)

// Metricsets / GCP services names
const (
	ServiceCompute       = "compute"
	ServicePubsub        = "pubsub"
	ServiceLoadBalancing = "loadbalancing"
	ServiceFirestore     = "firestore"
	ServiceStorage       = "storage"
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

	ECSCloudAccount   = "account"
	ECSCloudAccountID = "id"

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
