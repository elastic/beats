// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package googlecloud

const (
	MODULE_NAME                           = "googlecloud"
	MAX_TIME_INTERVAL_DATA_WINDOW_MINUTES = 5
)

const (
	SERVICE_COMPUTE   = "compute"
	SERVICE_PUBSUB    = "pubsub"
	SERVICE_FIRESTORE = "firestore"
	SERVICE_STORAGE   = "storage"
)

//Paths within the GCP monitoring.TimeSeries response, if converted to JSON, where you can find each ECS field required for the output event
const (
	JSON_PATH_ECS_AVAILABILITY_ZONE = "zone"
	JSON_PATH_ECS_ACCOUNT_ID        = "project_id"
	JSON_PATH_ECS_INSTANCE_ID       = "instance_id"
	JSON_PATH_ECS_INSTANCE_NAME     = "instance_name"
)

// ECS Fields that are being captured from responses
const (
	//Cloud fields https://www.elastic.co/guide/en/ecs/master/ecs-cloud.html
	ECS_CLOUD                   = "cloud"
	ECS_CLOUD_AVAILABILITY_ZONE = "availability_zone"
	ECS_CLOUD_ACCOUNT           = "account"
	ECS_CLOUD_ACCOUNT_ID        = "id"
	ECS_CLOUD_INSTANCE          = "instance"
	ECS_CLOUD_INSTANCE_ID       = "id"
	ECS_CLOUD_INSTANCE_NAME     = "name"
	ECS_CLOUD_MACHINE           = "machine"
	ECS_CLOUD_MACHINE_TYPE      = "type"
	ECS_CLOUD_PROVIDER          = "provider"
	ECS_CLOUD_REGION            = "region"
)

// Metadata keys used for events. They follow GCP structure.
const (
	//Stackdriver
	LABEL_METRICS       = "metrics"
	LABEL_RESOURCE      = "resource"
	LABEL_SYSTEM        = "system"
	LABEL_USER_METADATA = "metadata.user"
	KEY_TIMESTAMP       = "timestamp"

	// Compute
	LABEL_USER     = "user"
	LABEL_METADATA = "metadata"
)
