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
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func createEC2MockAPI(responseMap map[string]string) *httptest.Server {
	h := func(w http.ResponseWriter, r *http.Request) {
		if res, ok := responseMap[r.RequestURI]; ok {
			w.Write([]byte(res))
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}
	return httptest.NewServer(http.HandlerFunc(h))
}

func TestMain(m *testing.M) {
	logp.TestingSetup()
	code := m.Run()
	os.Exit(code)

}

func TestRetrieveAWSMetadataEC2(t *testing.T) {

	const (
		// not the best way to use a response template
		// but this should serve until we need to test
		// documents containing very different values
		accountIDDoc1        = "111111111111111"
		regionDoc1           = "us-east-1"
		availabilityZoneDoc1 = "us-east-1c"
		instanceIDDoc1       = "i-11111111"
		imageIDDoc1          = "ami-abcd1234"
		instanceTypeDoc1     = "t2.medium"

		instanceIDDoc2 = "i-22222222"

		templateDoc = `{
	  "accountId" : "%s",
	  "region" : "%s",
	  "availabilityZone" : "%s",
	  "instanceId" : "%s",
	  "imageId" : "%s",
	  "instanceType" : "%s",
	  "devpayProductCodes" : null,
	  "privateIp" : "10.0.0.1",	  
	  "version" : "2010-08-31",
	  "billingProducts" : null,
	  "pendingTime" : "2016-09-20T15:43:02Z",
	  "architecture" : "x86_64",
	  "kernelId" : null,
	  "ramdiskId" : null
	}`
	)

	sampleEC2Doc1 := fmt.Sprintf(
		templateDoc,
		accountIDDoc1,
		regionDoc1,
		availabilityZoneDoc1,
		instanceIDDoc1,
		imageIDDoc1,
		instanceTypeDoc1,
	)

	var testCases = []struct {
		testName           string
		ec2ResponseMap     map[string]string
		processorOverwrite bool
		previousEvent      common.MapStr

		expectedEvent common.MapStr
	}{
		{
			testName:           "all fields from processor",
			ec2ResponseMap:     map[string]string{ec2InstanceIdentityURI: sampleEC2Doc1},
			processorOverwrite: false,
			previousEvent:      common.MapStr{},
			expectedEvent: common.MapStr{
				"cloud": common.MapStr{
					"provider":          "aws",
					"account":           common.MapStr{"id": accountIDDoc1},
					"instance":          common.MapStr{"id": instanceIDDoc1},
					"machine":           common.MapStr{"type": instanceTypeDoc1},
					"image":             common.MapStr{"id": imageIDDoc1},
					"region":            regionDoc1,
					"availability_zone": availabilityZoneDoc1,
				},
			},
		},

		{
			testName:           "instanceId pre-informed, no overwrite",
			ec2ResponseMap:     map[string]string{ec2InstanceIdentityURI: sampleEC2Doc1},
			processorOverwrite: false,
			previousEvent: common.MapStr{
				"cloud": common.MapStr{
					"instance": common.MapStr{"id": instanceIDDoc2},
				},
			},
			expectedEvent: common.MapStr{
				"cloud": common.MapStr{
					"instance": common.MapStr{"id": instanceIDDoc2},
				},
			},
		},

		{
			// NOTE: In this case, add_cloud_metadata will overwrite cloud fields because
			// it won't detect cloud.provider as a cloud field. This is not the behavior we
			// expect and will find a better solution later in issue 11697.
			testName:           "only cloud.provider pre-informed, no overwrite",
			ec2ResponseMap:     map[string]string{ec2InstanceIdentityURI: sampleEC2Doc1},
			processorOverwrite: false,
			previousEvent: common.MapStr{
				"cloud.provider": "aws",
			},
			expectedEvent: common.MapStr{
				"cloud.provider": "aws",
				"cloud": common.MapStr{
					"provider":          "aws",
					"account":           common.MapStr{"id": accountIDDoc1},
					"instance":          common.MapStr{"id": instanceIDDoc1},
					"machine":           common.MapStr{"type": instanceTypeDoc1},
					"image":             common.MapStr{"id": imageIDDoc1},
					"region":            regionDoc1,
					"availability_zone": availabilityZoneDoc1,
				},
			},
		},

		{
			testName:           "all fields from processor, overwrite",
			ec2ResponseMap:     map[string]string{ec2InstanceIdentityURI: sampleEC2Doc1},
			processorOverwrite: true,
			previousEvent:      common.MapStr{},
			expectedEvent: common.MapStr{
				"cloud": common.MapStr{
					"provider":          "aws",
					"account":           common.MapStr{"id": accountIDDoc1},
					"instance":          common.MapStr{"id": instanceIDDoc1},
					"machine":           common.MapStr{"type": instanceTypeDoc1},
					"image":             common.MapStr{"id": imageIDDoc1},
					"region":            regionDoc1,
					"availability_zone": availabilityZoneDoc1,
				},
			},
		},

		{
			testName:           "instanceId pre-informed, overwrite",
			ec2ResponseMap:     map[string]string{ec2InstanceIdentityURI: sampleEC2Doc1},
			processorOverwrite: true,
			previousEvent: common.MapStr{
				"cloud": common.MapStr{
					"instance": common.MapStr{"id": instanceIDDoc2},
				},
			},
			expectedEvent: common.MapStr{
				"cloud": common.MapStr{
					"provider":          "aws",
					"account":           common.MapStr{"id": accountIDDoc1},
					"instance":          common.MapStr{"id": instanceIDDoc1},
					"machine":           common.MapStr{"type": instanceTypeDoc1},
					"image":             common.MapStr{"id": imageIDDoc1},
					"region":            regionDoc1,
					"availability_zone": availabilityZoneDoc1,
				},
			},
		},

		{
			testName:           "only cloud.provider pre-informed, overwrite",
			ec2ResponseMap:     map[string]string{ec2InstanceIdentityURI: sampleEC2Doc1},
			processorOverwrite: false,
			previousEvent: common.MapStr{
				"cloud.provider": "aws",
			},
			expectedEvent: common.MapStr{
				"cloud.provider": "aws",
				"cloud": common.MapStr{
					"provider":          "aws",
					"account":           common.MapStr{"id": accountIDDoc1},
					"instance":          common.MapStr{"id": instanceIDDoc1},
					"machine":           common.MapStr{"type": instanceTypeDoc1},
					"image":             common.MapStr{"id": imageIDDoc1},
					"region":            regionDoc1,
					"availability_zone": availabilityZoneDoc1,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			server := createEC2MockAPI(tc.ec2ResponseMap)
			defer server.Close()

			config, err := common.NewConfigFrom(map[string]interface{}{
				"host":      server.Listener.Addr().String(),
				"overwrite": tc.processorOverwrite,
			})
			if err != nil {
				t.Fatalf("error creating config from map: %s", err.Error())
			}

			cmp, err := New(config)
			if err != nil {
				t.Fatalf("error creating new metadata processor: %s", err.Error())
			}

			actual, err := cmp.Run(&beat.Event{Fields: tc.previousEvent})
			if err != nil {
				t.Fatalf("error running processor: %s", err.Error())
			}
			assert.Equal(t, tc.expectedEvent, actual.Fields)
		})
	}
}
