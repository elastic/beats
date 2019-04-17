// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package add_cloud_metadata

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

const ec2InstanceIdentityDocument = `{
  "devpayProductCodes" : null,
  "privateIp" : "10.0.0.1",
  "availabilityZone" : "us-east-1c",
  "accountId" : "111111111111111",
  "version" : "2010-08-31",
  "instanceId" : "i-11111111",
  "billingProducts" : null,
  "instanceType" : "t2.medium",
  "imageId" : "ami-6869aa05",
  "pendingTime" : "2016-09-20T15:43:02Z",
  "architecture" : "x86_64",
  "kernelId" : null,
  "ramdiskId" : null,
  "region" : "us-east-1"
}`

func initEC2TestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/2014-02-25/dynamic/instance-identity/document" {
			w.Write([]byte(ec2InstanceIdentityDocument))
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
}

func TestRetrieveAWSMetadata(t *testing.T) {
	logp.TestingSetup()

	server := initEC2TestServer()
	defer server.Close()

	config, err := common.NewConfigFrom(map[string]interface{}{
		"host":      server.Listener.Addr().String(),
		"overwrite": false,
	})
	if err != nil {
		t.Fatal(err)
	}

	p, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		fields          common.MapStr
		expectedResults common.MapStr
	}{
		{
			common.MapStr{},
			common.MapStr{
				"cloud": common.MapStr{
					"provider": "aws",
					"instance": common.MapStr{
						"id": "i-11111111",
					},
					"machine": common.MapStr{
						"type": "t2.medium",
					},
					"region":            "us-east-1",
					"availability_zone": "us-east-1c",
				},
			},
		},
		{
			common.MapStr{
				"cloud": common.MapStr{
					"instance": common.MapStr{
						"id": "i-000",
					},
				},
			},
			common.MapStr{
				"cloud": common.MapStr{
					"instance": common.MapStr{
						"id": "i-000",
					},
				},
			},
		},
		{
			common.MapStr{
				"provider": "aws",
			},
			common.MapStr{
				"provider": "aws",
				"cloud": common.MapStr{
					"provider": "aws",
					"instance": common.MapStr{
						"id": "i-11111111",
					},
					"machine": common.MapStr{
						"type": "t2.medium",
					},
					"region":            "us-east-1",
					"availability_zone": "us-east-1c",
				},
			},
		},
		{
			common.MapStr{
				"cloud.provider": "aws",
			},
			// NOTE: In this case, add_cloud_metadata will overwrite cloud fields because
			// it won't detect cloud.provider as a cloud field. This is not the behavior we
			// expect and will find a better solution later in issue 11697.
			common.MapStr{
				"cloud.provider": "aws",
				"cloud": common.MapStr{
					"provider": "aws",
					"instance": common.MapStr{
						"id": "i-11111111",
					},
					"machine": common.MapStr{
						"type": "t2.medium",
					},
					"region":            "us-east-1",
					"availability_zone": "us-east-1c",
				},
			},
		},
	}

	for _, c := range cases {
		actual, err := p.Run(&beat.Event{Fields: c.fields})
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, c.expectedResults, actual.Fields)
	}
}

func TestRetrieveAWSMetadataOverwriteTrue(t *testing.T) {
	logp.TestingSetup()

	server := initEC2TestServer()
	defer server.Close()

	config, err := common.NewConfigFrom(map[string]interface{}{
		"host":      server.Listener.Addr().String(),
		"overwrite": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	p, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		fields          common.MapStr
		expectedResults common.MapStr
	}{
		{
			common.MapStr{},
			common.MapStr{
				"cloud": common.MapStr{
					"provider": "aws",
					"instance": common.MapStr{
						"id": "i-11111111",
					},
					"machine": common.MapStr{
						"type": "t2.medium",
					},
					"region":            "us-east-1",
					"availability_zone": "us-east-1c",
				},
			},
		},
		{
			common.MapStr{
				"cloud": common.MapStr{
					"instance": common.MapStr{
						"id": "i-000",
					},
				},
			},
			common.MapStr{
				"cloud": common.MapStr{
					"provider": "aws",
					"instance": common.MapStr{
						"id": "i-11111111",
					},
					"machine": common.MapStr{
						"type": "t2.medium",
					},
					"region":            "us-east-1",
					"availability_zone": "us-east-1c",
				},
			},
		},
		{
			common.MapStr{
				"cloud.provider": "aws",
			},
			common.MapStr{
				"cloud.provider": "aws",
				"cloud": common.MapStr{
					"provider": "aws",
					"instance": common.MapStr{
						"id": "i-11111111",
					},
					"machine": common.MapStr{
						"type": "t2.medium",
					},
					"region":            "us-east-1",
					"availability_zone": "us-east-1c",
				},
			},
		},
	}

	for _, c := range cases {
		actual, err := p.Run(&beat.Event{Fields: c.fields})
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, c.expectedResults, actual.Fields)
	}
}
