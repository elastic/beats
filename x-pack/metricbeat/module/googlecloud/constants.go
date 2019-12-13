// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package googlecloud

const (
	ModuleName                       = "googlecloud"
	MaxTimeIntervalDataWindowMinutes = 5
)

const (
	ServiceCompute   = "compute"
	ServicePubsub    = "pubsub"
	ServiceFirestore = "firestore"
	ServiceStorage   = "storage"
)

//Paths within the GCP monitoring.TimeSeries response, if converted to JSON, where you can find each ECS field required for the output event
const (
	TimeSeriesResponsePathForECSAvailabilityZone = "zone"
	TimeSeriesResponsePathForECSAccountId        = "project_id"
	TimeSeriesResponsePathForECSInstanceId       = "instance_id"
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
	ECSCloudAccountId = "id"

	ECSCloudInstance        = "instance"
	ECSCloudInstanceKey     = ECSCloud + "." + ECSCloudInstance
	ECSCloudInstanceId      = "id"
	ECSCloudInstanceIdKey   = ECSCloudInstanceKey + "." + ECSCloudInstanceId
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
