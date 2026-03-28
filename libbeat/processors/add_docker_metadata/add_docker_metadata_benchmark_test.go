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

//go:build linux || darwin || windows

package add_docker_metadata

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-autodiscover/docker"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func BenchmarkAddDockerMetadata(b *testing.B) {
	cfg, err := conf.NewConfigFrom(map[string]interface{}{
		"match_fields": []string{"container.id"},
	})
	if err != nil {
		b.Fatal(err)
	}

	p, err := buildDockerMetadataProcessor(logptest.NewTestingLogger(b, ""), cfg, MockWatcherFactory(
		map[string]*docker.Container{
			"abc123": {
				ID:    "abc123def456",
				Image: "myrepo/myimage:latest",
				Name:  "my-container",
				Labels: map[string]string{
					"app":     "myapp",
					"version": "v1.2.3",
					"env":     "production",
				},
			},
		}, nil))
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event := &beat.Event{
			Fields: mapstr.M{
				"container": mapstr.M{"id": "abc123"},
				"message":   "test log line",
			},
		}
		_, err := p.Run(event)
		if err != nil {
			b.Fatal(err)
		}
	}
}
