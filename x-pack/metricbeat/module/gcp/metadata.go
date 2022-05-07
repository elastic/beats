// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"context"
	"time"

	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// MetadataService must be implemented by GCP services that requires non out-of-the box code that is not fulfil by the Stackdriver
// metricset. For example, Compute instance labels.
type MetadataService interface {
	MetadataCollector
	Identity
}

// MetadataCollector must be implemented by services that has special code needs that aren't fulfilled by the Stackdriver
// metricset to collect their labels (most of them)
type MetadataCollector interface {

	// Metadata returns an object with perfectly formatted labels and ECS fields ready to attach to an output event in
	//its "labels" key. For example, Compute labels looks like this. Other services may have a slightly different
	//structure. Check constants.go file for reference:
	//
	// {
	//    "metadata":{
	//        "key":"value"
	//		  "user": {
	//		    "key": "value"
	//		  }
	//    },
	//    "system":{
	//        "key":"value"
	//    },
	//    "metrics":{
	//        "key":"value"
	//    },
	//    "user":{
	//        "key":"value"
	//    },
	// }
	// Because some of them will be ECS fields, the second returned MapStr are those ECS fields.
	Metadata(ctx context.Context, in *monitoringpb.TimeSeries) (MetadataCollectorData, error)
}

// MetadataCollectorInputData is a "container" of input data commonly needed for the GCP service's metadata collectors
type MetadataCollectorInputData struct {
	TimeSeries *monitoringpb.TimeSeries
	ProjectID  string
	Zone       string
	Region     string
	Point      *monitoringpb.Point
	Timestamp  *time.Time
}

// MetadataCollectorData contains the set of ECS and normal labels that we extract from GCP services
type MetadataCollectorData struct {
	Labels mapstr.M
	ECS    mapstr.M
}

// Identity must be implemented by GCP services that can add some short of data to group their metrics (like instance
// id on Compute or topic in PubSub)
type Identity interface {

	// ID returns a unique identifier to group many metrics into a single event
	ID(ctx context.Context, in *MetadataCollectorInputData) (string, error)
}
