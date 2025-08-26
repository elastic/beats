// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"errors"
	"sync"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/management/status"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// statusReporterHelperMock is a thread-safe mock of a status reporter that
// behaves like StatusReporterHelper
type statusReporterHelperMock struct {
	mu       sync.Mutex
	statuses []mgmtStatusUpdate
	current  status.Status
}

type mgmtStatusUpdate struct {
	status status.Status
	msg    string
}

func (r *statusReporterHelperMock) getStatuses() []mgmtStatusUpdate {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := make([]mgmtStatusUpdate, len(r.statuses))
	copy(s, r.statuses)
	return s
}

func (r *statusReporterHelperMock) UpdateStatus(s status.Status, msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Imitate behavior of statusReporterHelper. Only record if the new status is different.
	if s != r.current {
		r.current = s
		r.statuses = append(r.statuses, mgmtStatusUpdate{status: s, msg: msg})
	}
}

func TestRegionSelection(t *testing.T) {
	tests := []struct {
		name       string
		queueURL   string
		regionName string
		endpoint   string
		want       string
		wantErr    error
	}{
		{
			name:     "amazonaws.com_domain_with_blank_endpoint",
			queueURL: "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs",
			want:     "us-east-1",
		},
		{
			name:       "amazonaws.com_domain_with_region_override",
			queueURL:   "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs",
			regionName: "us-east-2",
			want:       "us-east-2",
		},
		{
			name:     "abc.xyz_and_domain_with_matching_endpoint",
			queueURL: "https://sqs.us-east-1.abc.xyz/627959692251/test-s3-logs",
			endpoint: "abc.xyz",
			want:     "us-east-1",
		},
		{
			name:       "abc.xyz_with_region_override",
			queueURL:   "https://sqs.us-east-1.abc.xyz/627959692251/test-s3-logs",
			regionName: "us-west-3",
			want:       "us-west-3",
		},
		{
			name:     "abc.xyz_and_domain_with_matching_endpoint_and_scheme",
			queueURL: "https://sqs.us-east-1.abc.xyz/627959692251/test-s3-logs",
			endpoint: "https://abc.xyz",
			want:     "us-east-1",
		},
		{
			name:     "abc.xyz_and_domain_with_matching_url_endpoint",
			queueURL: "https://sqs.us-east-1.abc.xyz/627959692251/test-s3-logs",
			endpoint: "https://s3.us-east-1.abc.xyz",
			want:     "us-east-1",
		},
		{
			name:     "abc.xyz_and_no_region_term",
			queueURL: "https://sqs.abc.xyz/627959692251/test-s3-logs",
			wantErr:  errBadQueueURL,
		},
		{
			name:     "vpce_endpoint",
			queueURL: "https://vpce-test.sqs.us-east-2.vpce.amazonaws.com/12345678912/sqs-queue",
			want:     "us-east-2",
		},
		{
			name:       "vpce_endpoint_with_region_override",
			queueURL:   "https://vpce-test.sqs.us-east-2.vpce.amazonaws.com/12345678912/sqs-queue",
			regionName: "us-west-1",
			want:       "us-west-1",
		},
		{
			name:     "vpce_endpoint_with_endpoint",
			queueURL: "https://vpce-test.sqs.us-east-1.vpce.amazonaws.com/12345678912/sqs-queue",
			endpoint: "amazonaws.com",
			want:     "us-east-1",
		},
		{
			name:     "non_aws_vpce_with_endpoint",
			queueURL: "https://vpce-test.sqs.us-east-1.vpce.abc.xyz/12345678912/sqs-queue",
			endpoint: "abc.xyz",
			want:     "us-east-1",
		},
		{
			name:     "non_aws_vpce_without_endpoint",
			queueURL: "https://vpce-test.sqs.us-east-1.vpce.abc.xyz/12345678912/sqs-queue",
			want:     "us-east-1",
		},
		{
			name:       "non_aws_vpce_with_region_override",
			queueURL:   "https://vpce-test.sqs.us-east-1.vpce.abc.xyz/12345678912/sqs-queue",
			regionName: "us-west-1",
			want:       "us-west-1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := config{
				QueueURL:   test.queueURL,
				RegionName: test.regionName,
				AWSConfig:  awscommon.ConfigAWS{Endpoint: test.endpoint},
			}
			in := newSQSReaderInput(config, awssdk.Config{})
			inputCtx := v2.Context{
				Logger:          logp.NewLogger("awss3_test"),
				ID:              "test_id",
				MetricsRegistry: monitoring.NewRegistry(),
			}

			in.status = &statusReporterHelperMock{}
			// Run setup and verify that it put the correct region in awsConfig.Region
			err := in.setup(inputCtx, &fakePipeline{})
			in.cleanup()
			got := in.awsConfig.Region // The region passed into the AWS API
			if !errors.Is(err, test.wantErr) {
				t.Errorf("unexpected error: got:%v want:%v", err, test.wantErr)
			}
			if got != test.want {
				t.Errorf("unexpected result: got:%q want:%q", got, test.want)
			}
		})
	}
}
