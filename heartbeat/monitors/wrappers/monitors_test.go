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

package wrappers

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/heartbeat/eventext"
	"github.com/elastic/beats/heartbeat/monitors/jobs"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/mapval"
	"github.com/elastic/beats/libbeat/testing/mapvaltest"
)

func TestWrapCommon(t *testing.T) {
	var simpleJob jobs.Job = func(event *beat.Event) ([]jobs.Job, error) {
		eventext.MergeEventFields(event, common.MapStr{"simple": "job"})
		return nil, nil
	}
	simpleJobValidator := mapval.MustCompile(mapval.Map{"simple": "job"})

	var errorJob jobs.Job = func(event *beat.Event) ([]jobs.Job, error) {
		return nil, fmt.Errorf("myerror")
	}
	errorJobValidator := mapval.MustCompile(mapval.Map{
		"error": mapval.Map{
			"message": "myerror",
			"type":    "io",
		},
	})

	type fields struct {
		id   string
		name string
		typ  string
	}

	commonFieldsValidator := func(f fields, status string) mapval.Validator {
		return mapval.MustCompile(mapval.Map{
			"monitor": mapval.Map{
				"duration.us": mapval.IsDuration,
				"id":          f.id,
				"name":        f.name,
				"type":        f.typ,
				"status":      status,
			},
		})
	}

	testFields := fields{"myid", "myname", "mytyp"}

	tests := []struct {
		name   string
		fields fields
		jobs   []jobs.Job
		want   []mapval.Validator
	}{
		{
			"simple",
			testFields,
			[]jobs.Job{simpleJob},
			[]mapval.Validator{
				mapval.Strict(mapval.Compose(
					simpleJobValidator,
					commonFieldsValidator(testFields, "up"),
				)),
			},
		},
		{
			"job error",
			testFields,
			[]jobs.Job{errorJob},
			[]mapval.Validator{
				mapval.Strict(mapval.Compose(
					errorJobValidator,
					commonFieldsValidator(testFields, "down"),
				)),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := WrapCommon(tt.jobs, tt.fields.id, tt.fields.name, tt.fields.typ)

			results, err := jobs.ExecJobsAndConts(t, wrapped)
			assert.NoError(t, err)

			for idx, r := range results {
				mapvaltest.Test(t, tt.want[idx], r.Fields)
			}
		})
	}
}
