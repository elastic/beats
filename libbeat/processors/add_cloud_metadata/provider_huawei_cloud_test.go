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

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func initHuaweiCloudTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/openstack/latest/meta_data.json" {
			w.Write([]byte(`{
				"random_seed": "CWIZtYK4y5pzMtShTtCKx16qB1DsA/2kL0US4u1fHxedODNr7gos4RgdE/z9eHucnltnlJfDY1remfGL60yzTsvEIWPdECOpPaJm1edIYQaUvQzdeQwKcOQAHjUP5wLQzGA3j3Pw10p7u+M7glHEwNRoEY1WsbVYwzyOOkBnqb+MJ1aOhiRnfNtHOxjLNBSDvjHaQZzoHL+1YNAxDYFezE83nE2m3ciVwZO7xWpdKDQ+W5hYBUsYAWODRMOYqIR/5ZLsfAfxE2DhK+NvuMyJ5yjO+ObQf0DN5nRUSrM5ajs84UVMr9ylJuT78ckh83CLSttsjzXJ+sr07ZFsB6/6NABzziFL7Xn8z/mEBVmFXBiBgg7KcWSoH756w42VSdUezwTy9lW0spRmdvNBKV/PzrYyy0FMiGXXZwMOCyBD05CBRJlsPorwxZLlfRVmNvsTuMYB8TG3UUbFhoR8Bd5en+EC3ncH3QIUDWn0oVg28BVjWe5rADVQLX1h83ti6GD08YUGaxoNPXnJLZfiaucSacby2mG31xysxd8Tg0qPRq7744a1HPVryuauWR9pF0+qDmtskhenxK0FR+TQ4w0fRxTigteBsXx1pQu0iz+B8rP68uokU2faCC2IMHY2Tf9RPCe6Eef0/DdQhBft88PuJLwq52o/0qZ/n9HFL6LdgCU=",
				"uuid": "37da9890-8289-4c58-ba34-a8271c4a8216",
				"availability_zone": "cn-east-2b",
				"enterprise_project_id": "0",
				"launch_index": 0,
				"instance_type": "c3.large.2",
				"meta": {
				  "os_bit": "64",
				  "image_name": "CentOS 7.4",
				  "vpc_id": "6dad7f50-db1d-4cce-b095-d27bc837d4bb"
				},
				"region_id": "cn-east-2",
				"project_id": "c09b8baf28b845a9b53ed37575cfd61f",
				"name": "hwdev-test-1"
			}`))
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
}

func TestRetrieveHuaweiCloudMetadata(t *testing.T) {
	logp.TestingSetup()

	server := initHuaweiCloudTestServer()
	defer server.Close()

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"providers": []string{"huawei"},
		"host":      server.Listener.Addr().String(),
	})

	if err != nil {
		t.Fatal(err)
	}

	p, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	actual, err := p.Run(&beat.Event{Fields: mapstr.M{}})
	if err != nil {
		t.Fatal(err)
	}

	expected := mapstr.M{
		"cloud": mapstr.M{
			"provider": "huawei",
			"instance": mapstr.M{
				"id": "37da9890-8289-4c58-ba34-a8271c4a8216",
			},
			"region":            "cn-east-2",
			"availability_zone": "cn-east-2b",
			"service": mapstr.M{
				"name": "ECS",
			},
		},
	}
	assert.Equal(t, expected, actual.Fields)
}
