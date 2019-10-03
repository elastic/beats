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

// +build integration

package docker

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/autodiscover/template"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	dk "github.com/elastic/beats/libbeat/tests/docker"
)

// Test docker start emits an autodiscover event
func TestDockerStart(t *testing.T) {
	d, err := dk.NewClient()
	if err != nil {
		t.Fatal(err)
	}

	UUID, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}
	bus := bus.New("test")
	config := defaultConfig()
	config.CleanupTimeout = 0

	s := &template.MapperSettings{nil, nil}
	config.Templates = *s
	provider, err := AutodiscoverBuilder(bus, UUID, common.MustNewConfigFrom(config))
	if err != nil {
		t.Fatal(err)
	}

	provider.Start()
	defer provider.Stop()

	listener := bus.Subscribe()
	defer listener.Stop()

	// Start
	cmd := []string{"echo", "Hi!"}
	labels := map[string]string{"label": "foo", "label.child": "bar"}
	ID, err := d.ContainerStart("busybox", cmd, labels)
	if err != nil {
		t.Fatal(err)
	}
	checkEvent(t, listener, true)

	// Kill
	d.ContainerKill(ID)
	checkEvent(t, listener, false)
}

func getValue(e bus.Event, key string) interface{} {
	val, err := common.MapStr(e).GetValue(key)
	if err != nil {
		return nil
	}
	return val
}

func checkEvent(t *testing.T, listener bus.Listener, start bool) {
	for {
		select {
		case e := <-listener.Events():
			// Ignore any other container
			if getValue(e, "docker.container.image") != "busybox" {
				continue
			}
			if start {
				assert.Equal(t, getValue(e, "start"), true)
				assert.Nil(t, getValue(e, "stop"))
			} else {
				assert.Equal(t, getValue(e, "stop"), true)
				assert.Nil(t, getValue(e, "start"))
			}
			assert.Equal(t, getValue(e, "container.image.name"), "busybox")
			// labels.dedot=true by default
			assert.Equal(t,
				common.MapStr{
					"label": common.MapStr{
						"value": "foo",
						"child": "bar",
					},
				},
				getValue(e, "container.labels"),
			)
			assert.NotNil(t, getValue(e, "container.id"))
			assert.NotNil(t, getValue(e, "container.name"))
			assert.NotNil(t, getValue(e, "host"))
			assert.Equal(t, getValue(e, "docker.container.id"), getValue(e, "meta.container.id"))
			assert.Equal(t, getValue(e, "docker.container.name"), getValue(e, "meta.container.name"))
			assert.Equal(t, getValue(e, "docker.container.image"), getValue(e, "meta.container.image.name"))
			return

		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for provider events")
			return
		}
	}
}
