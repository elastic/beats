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
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/heartbeat/eventext"
	"github.com/elastic/beats/heartbeat/monitors/jobs"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/mapval"
)

type fields struct {
	id   string
	name string
	typ  string
}

type testDef struct {
	name   string
	fields fields
	jobs   []jobs.Job
	want   []mapval.Validator
}

func testCommonWrap(t *testing.T, tt testDef) {
	t.Run(tt.name, func(t *testing.T) {
		wrapped := WrapCommon(tt.jobs, tt.fields.id, tt.fields.name, tt.fields.typ)

		results, err := jobs.ExecJobsAndConts(t, wrapped)
		assert.NoError(t, err)

		for idx, r := range results {
			t.Run(fmt.Sprintf("result at index %d", idx), func(t *testing.T) {
				mapval.Test(t, mapval.Strict(tt.want[idx]), r.Fields)
			})
		}
	})
}

func TestSimpleJob(t *testing.T) {
	fields := fields{"myid", "myname", "mytyp"}
	testCommonWrap(t, testDef{
		"simple",
		fields,
		[]jobs.Job{makeURLJob(t, "tcp://foo.com:80")},
		[]mapval.Validator{
			mapval.Compose(
				urlValidator(t, "tcp://foo.com:80"),
				mapval.MustCompile(mapval.Map{
					"monitor": mapval.Map{
						"duration.us": mapval.IsDuration,
						"id":          fields.id,
						"name":        fields.name,
						"type":        fields.typ,
						"status":      "up",
						"check_group": mapval.IsString,
					},
				}),
				summaryValidator(1, 0),
			)},
	})
}

func TestErrorJob(t *testing.T) {
	fields := fields{"myid", "myname", "mytyp"}

	errorJob := func(event *beat.Event) ([]jobs.Job, error) {
		return nil, fmt.Errorf("myerror")
	}

	errorJobValidator := mapval.Compose(
		mapval.MustCompile(mapval.Map{"error": mapval.Map{"message": "myerror", "type": "io"}}),
		mapval.MustCompile(mapval.Map{
			"monitor": mapval.Map{
				"duration.us": mapval.IsDuration,
				"id":          fields.id,
				"name":        fields.name,
				"type":        fields.typ,
				"status":      "down",
				"check_group": mapval.IsString,
			},
		}),
	)

	testCommonWrap(t, testDef{
		"job error",
		fields,
		[]jobs.Job{errorJob},
		[]mapval.Validator{
			mapval.Compose(
				errorJobValidator,
				summaryValidator(0, 1),
			)},
	})
}

func TestMultiJobNoConts(t *testing.T) {
	fields := fields{"myid", "myname", "mytyp"}

	uniqScope := mapval.ScopedIsUnique()

	validator := func(u string) mapval.Validator {
		return mapval.Compose(
			urlValidator(t, u),
			mapval.MustCompile(mapval.Map{
				"monitor": mapval.Map{
					"duration.us": mapval.IsDuration,
					"id":          uniqScope.IsUniqueTo("id"),
					"name":        fields.name,
					"type":        fields.typ,
					"status":      "up",
					"check_group": uniqScope.IsUniqueTo("check_group"),
				},
			}),
			summaryValidator(1, 0),
		)
	}

	testCommonWrap(t, testDef{
		"multi-job",
		fields,
		[]jobs.Job{makeURLJob(t, "http://foo.com"), makeURLJob(t, "http://bar.com")},
		[]mapval.Validator{validator("http://foo.com"), validator("http://bar.com")},
	})
}

func TestMultiJobConts(t *testing.T) {
	fields := fields{"myid", "myname", "mytyp"}

	uniqScope := mapval.ScopedIsUnique()

	makeContJob := func(t *testing.T, u string) jobs.Job {
		return func(event *beat.Event) ([]jobs.Job, error) {
			eventext.MergeEventFields(event, common.MapStr{"cont": "1st"})
			u, err := url.Parse(u)
			require.NoError(t, err)
			eventext.MergeEventFields(event, common.MapStr{"url": URLFields(u)})
			return []jobs.Job{
				func(event *beat.Event) ([]jobs.Job, error) {
					eventext.MergeEventFields(event, common.MapStr{"cont": "2nd"})
					eventext.MergeEventFields(event, common.MapStr{"url": URLFields(u)})
					return nil, nil
				},
			}, nil
		}
	}

	contJobValidator := func(u string, msg string) mapval.Validator {
		return mapval.Compose(
			urlValidator(t, u),
			mapval.MustCompile(mapval.Map{"cont": msg}),
			mapval.MustCompile(mapval.Map{
				"monitor": mapval.Map{
					"duration.us": mapval.IsDuration,
					"id":          uniqScope.IsUniqueTo(u),
					"name":        fields.name,
					"type":        fields.typ,
					"status":      "up",
					"check_group": uniqScope.IsUniqueTo(u),
				},
			}),
		)
	}

	testCommonWrap(t, testDef{
		"multi-job-continuations",
		fields,
		[]jobs.Job{makeContJob(t, "http://foo.com"), makeContJob(t, "http://bar.com")},
		[]mapval.Validator{
			contJobValidator("http://foo.com", "1st"),
			mapval.Compose(
				contJobValidator("http://foo.com", "2nd"),
				summaryValidator(2, 0),
			),
			contJobValidator("http://bar.com", "1st"),
			mapval.Compose(
				contJobValidator("http://bar.com", "2nd"),
				summaryValidator(2, 0),
			),
		},
	})
}

func makeURLJob(t *testing.T, u string) jobs.Job {
	parsed, err := url.Parse(u)
	require.NoError(t, err)
	return func(event *beat.Event) (i []jobs.Job, e error) {
		eventext.MergeEventFields(event, common.MapStr{"url": URLFields(parsed)})
		return nil, nil
	}
}

func urlValidator(t *testing.T, u string) mapval.Validator {
	parsed, err := url.Parse(u)
	require.NoError(t, err)
	return mapval.MustCompile(mapval.Map{"url": mapval.Map(URLFields(parsed))})
}

// This duplicates hbtest.SummaryChecks to avoid an import cycle.
// It could be refactored out, but it just isn't worth it.
func summaryValidator(up int, down int) mapval.Validator {
	return mapval.MustCompile(mapval.Map{
		"summary": mapval.Map{
			"up":   uint16(up),
			"down": uint16(down),
		},
	})
}
