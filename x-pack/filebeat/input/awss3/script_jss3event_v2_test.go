// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/logp"
)

const (
	header = `function parse(n) {`
	footer = `}`
)

var log = logp.NewLogger("test")

func TestJSS3EventV2(t *testing.T) {
	logp.TestingSetup()

	source := `
	var evts = [];
	var evt = new S3EventV2();
	evt.SetAWSRegion("region");
	evt.SetProvider("provider");
	evt.SetEventName("name");
	evt.SetEventSource("source");
	evt.SetS3BucketName("bucket");
	evt.SetS3BucketARN("arn");
	evt.SetS3ObjectKey("key");
	evts.push(evt);
	return evts;
	`

	p, err := newScriptFromConfig(log, &scriptConfig{Source: header + source + footer})
	if err != nil {
		t.Fatal(err)
	}

	evts, err := p.run(`{}`)
	require.NoError(t, err)
	require.Equal(t, 1, len(evts))

	exp := s3EventV2{
		AWSRegion:   "region",
		Provider:    "provider",
		EventName:   "name",
		EventSource: "source",
	}
	exp.S3.Bucket.Name = "bucket"
	exp.S3.Bucket.ARN = "arn"
	exp.S3.Object.Key = "key"

	assert.EqualValues(t, exp, evts[0])
}
