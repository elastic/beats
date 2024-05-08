// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	aws "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
)

func TestGetProviderFromDomain(t *testing.T) {
	tests := []struct {
		endpoint string
		override string
		want     string
	}{
		{endpoint: "", override: "", want: "aws"},
		{endpoint: "c2s.ic.gov", want: "aws"},
		{endpoint: "abc.com", override: "abc", want: "abc"},
		{endpoint: "oraclecloud.com", override: "xyz", want: "xyz"},
		{endpoint: "amazonaws.com", want: "aws"},
		{endpoint: "c2s.sgov.gov", want: "aws"},
		{endpoint: "c2s.ic.gov", want: "aws"},
		{endpoint: "amazonaws.com.cn", want: "aws"},
		{endpoint: "https://backblazeb2.com", want: "backblaze"},
		{endpoint: "https://1234567890.r2.cloudflarestorage.com", want: "cloudflare"},
		{endpoint: "https://wasabisys.com", want: "wasabi"},
		{endpoint: "https://digitaloceanspaces.com", want: "digitalocean"},
		{endpoint: "https://dream.io", want: "dreamhost"},
		{endpoint: "https://scw.cloud", want: "scaleway"},
		{endpoint: "https://googleapis.com", want: "gcp"},
		{endpoint: "https://cloud.it", want: "arubacloud"},
		{endpoint: "https://linodeobjects.com", want: "linode"},
		{endpoint: "https://vultrobjects.com", want: "vultr"},
		{endpoint: "https://appdomain.cloud", want: "ibm"},
		{endpoint: "https://aliyuncs.com", want: "alibaba"},
		{endpoint: "https://oraclecloud.com", want: "oracle"},
		{endpoint: "https://exo.io", want: "exoscale"},
		{endpoint: "https://upcloudobjects.com", want: "upcloud"},
		{endpoint: "https://ilandcloud.com", want: "iland"},
		{endpoint: "https://zadarazios.com", want: "zadara"},
	}

	for _, test := range tests {
		assert.Equal(t, test.want, getProviderFromDomain(test.endpoint, test.override),
			"for endpoint=%q and override=%q", test.endpoint, test.override)
	}
}

func TestGetRegionFromQueueURL(t *testing.T) {
	tests := []struct {
		name     string
		queueURL string
		endpoint string
		deflt    string
		want     string
		wantErr  error
	}{
		{
			name:     "amazonaws.com_domain_with_blank_endpoint",
			queueURL: "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs",
			want:     "us-east-1",
		},
		{
			name:     "abc.xyz_and_domain_with_matching_endpoint",
			queueURL: "https://sqs.us-east-1.abc.xyz/627959692251/test-s3-logs",
			endpoint: "abc.xyz",
			want:     "us-east-1",
		},
		{
			name:     "abc.xyz_and_domain_with_blank_endpoint",
			queueURL: "https://sqs.us-east-1.abc.xyz/627959692251/test-s3-logs",
			wantErr:  errBadQueueURL,
		},
		{
			name:     "abc.xyz_and_domain_with_different_endpoint",
			queueURL: "https://sqs.us-east-1.abc.xyz/627959692251/test-s3-logs",
			endpoint: "googlecloud.com",
			wantErr:  errBadQueueURL,
		},
		{
			name:     "mismatch_regions_no_default",
			queueURL: "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs",
			deflt:    "",
			want:     "us-east-1",
		},
		{
			name:     "mismatch_regions",
			queueURL: "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs",
			deflt:    "ap-west-1",
			want:     "ap-west-1",
			wantErr:  regionMismatchError{queueURLRegion: "us-east-1", defaultRegion: "ap-west-1"},
		},
		{
			name:     "localstack",
			queueURL: "http://localhost:4566/000000000000/filebeat-s3-integtest-d9clk9",
			deflt:    "localstack",
			want:     "localstack",
		},
		{
			name:     "localstack_sns",
			queueURL: "http://localhost:4566/000000000000/filebeat-s3-integtest-sns-d9clk9",
			deflt:    "localstack_sns",
			want:     "localstack_sns",
		},
		{
			name:     "invalid_queue_url",
			queueURL: ":foo",
			wantErr:  errors.New(":foo is not a valid URL"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := getRegionFromQueueURL(test.queueURL, test.endpoint, test.deflt)
			if !sameError(err, test.wantErr) {
				t.Errorf("unexpected error: got:%v want:%v", err, test.wantErr)
			}
			if got != test.want {
				t.Errorf("unexpected result: got:%q want:%q", got, test.want)
			}
		})
	}
}

func sameError(a, b error) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil, b == nil:
		return false
	default:
		return a.Error() == b.Error()
	}
}
