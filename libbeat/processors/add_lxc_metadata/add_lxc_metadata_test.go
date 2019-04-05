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

package add_lxc_metadata

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func init() {
	// Stub out the procfs.
	processCgroupPaths = func(_ string, pid int) (map[string]string, error) {
		switch pid {
		case 1000:
			return map[string]string{
				"cpu": "/lxc/125/ns",
			}, nil
		case 2000:
			return map[string]string{
				"memory": "/user.slice/svc.slice",
			}, nil
		case 3000:
			// Parser error (hopefully this never happens).
			return nil, fmt.Errorf("cgroup parse failure")
		default:
			return nil, os.ErrNotExist
		}
	}
}

func TestMatchPIDs(t *testing.T) {
	p, err := New(common.NewConfig())
	assert.NoError(t, err, "initializing add_lxc_metadata processor")

	lxcData := common.MapStr{}
	lxcData.Put("container.id", "125")

	t.Run("pid is not containerized", func(t *testing.T) {
		fields := common.MapStr{}
		fields.Put("process.pid", 2000)
		fields.Put("process.ppid", 1000)

		expected := common.MapStr{}
		expected.DeepUpdate(fields)

		result, err := p.Run(&beat.Event{Fields: fields})
		assert.NoError(t, err, "processing an event")
		assert.EqualValues(t, expected, result.Fields)
	})

	t.Run("pid does not exist", func(t *testing.T) {
		fields := common.MapStr{}
		fields.Put("process.pid", 9999)

		expected := common.MapStr{}
		expected.DeepUpdate(fields)

		result, err := p.Run(&beat.Event{Fields: fields})
		assert.NoError(t, err, "processing an event")
		assert.EqualValues(t, expected, result.Fields)
	})

	t.Run("pid is containerized", func(t *testing.T) {
		fields := common.MapStr{}
		fields.Put("process.pid", "1000")

		expected := common.MapStr{}
		expected.DeepUpdate(fields)
		expected.DeepUpdate(lxcData)

		result, err := p.Run(&beat.Event{Fields: fields})
		assert.NoError(t, err, "processing an event")
		assert.EqualValues(t, expected, result.Fields)
	})

	t.Run("pid exited and ppid is containerized", func(t *testing.T) {
		fields := common.MapStr{}
		fields.Put("process.pid", 9999)
		fields.Put("process.ppid", 1000)

		expected := common.MapStr{}
		expected.DeepUpdate(fields)
		expected.DeepUpdate(lxcData)

		result, err := p.Run(&beat.Event{Fields: fields})
		assert.NoError(t, err, "processing an event")
		assert.EqualValues(t, expected, result.Fields)
	})

	t.Run("cgroup error", func(t *testing.T) {
		fields := common.MapStr{}
		fields.Put("process.pid", 3000)

		expected := common.MapStr{}
		expected.DeepUpdate(fields)

		result, err := p.Run(&beat.Event{Fields: fields})
		assert.NoError(t, err, "processing an event")
		assert.EqualValues(t, expected, result.Fields)
	})
}
