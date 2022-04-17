// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package compute

import (
	"context"
	"errors"

	"github.com/menderesk/beats/v7/x-pack/metricbeat/module/gcp"
)

// ID returns a generated ID for a Compute resource based on its labels, projectID, zone, timestamp and instance ID
// It's purpose is to group metrics that share that ID (the ones on the same instance, basically)
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
