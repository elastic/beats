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

	"github.com/elastic/go-lookslike/isdef"

	"github.com/elastic/go-lookslike/validator"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/testslike"

	"github.com/elastic/beats/heartbeat/eventext"
	"github.com/elastic/beats/heartbeat/monitors/jobs"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

type fields struct {
	id   string
	name string
	typ  string
}

type testDef struct {
	name     string
	fields   fields
	jobs     []jobs.Job
	want     []validator.Validator
	metaWant []validator.Validator
}

func testCommonWrap(t *testing.T, tt testDef) {
	t.Run(tt.name, func(t *testing.T) {
		wrapped := WrapCommon(tt.jobs, tt.fields.id, tt.fields.name, tt.fields.typ)

		results, err := jobs.ExecJobsAndConts(t, wrapped)
		assert.NoError(t, err)

		require.Equal(t, len(results), len(tt.want), "Expected test def wants to correspond exactly to number results.")
		for idx, r := range results {
			t.Run(fmt.Sprintf("result at index %d", idx), func(t *testing.T) {
				want := tt.want[idx]
				testslike.Test(t, lookslike.Strict(want), r.Fields)

				if tt.metaWant != nil {
					metaWant := tt.metaWant[idx]
					testslike.Test(t, lookslike.Strict(metaWant), r.Meta)
				}
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
		[]validator.Validator{
			lookslike.Compose(
				urlValidator(t, "tcp://foo.com:80"),
				lookslike.MustCompile(map[string]interface{}{
					"monitor": map[string]interface{}{
						"duration.us": isdef.IsDuration,
						"id":          fields.id,
						"name":        fields.name,
						"type":        fields.typ,
						"status":      "up",
						"check_group": isdef.IsString,
					},
				}),
				summaryValidator(1, 0),
			)},
		nil,
	})
}

func TestErrorJob(t *testing.T) {
	fields := fields{"myid", "myname", "mytyp"}

	errorJob := func(event *beat.Event) ([]jobs.Job, error) {
		return nil, fmt.Errorf("myerror")
	}

	errorJobValidator := lookslike.Compose(
		lookslike.MustCompile(map[string]interface{}{"error": map[string]interface{}{"message": "myerror", "type": "io"}}),
		lookslike.MustCompile(map[string]interface{}{
			"monitor": map[string]interface{}{
				"duration.us": isdef.IsDuration,
				"id":          fields.id,
				"name":        fields.name,
				"type":        fields.typ,
				"status":      "down",
				"check_group": isdef.IsString,
			},
		}),
	)

	testCommonWrap(t, testDef{
		"job error",
		fields,
		[]jobs.Job{errorJob},
		[]validator.Validator{
			lookslike.Compose(
				errorJobValidator,
				summaryValidator(0, 1),
			)},
		nil,
	})
}

func TestMultiJobNoConts(t *testing.T) {
	fields := fields{"myid", "myname", "mytyp"}

	uniqScope := isdef.ScopedIsUnique()

	validatorMaker := func(u string) validator.Validator {
		return lookslike.Compose(
			urlValidator(t, u),
			lookslike.MustCompile(map[string]interface{}{
				"monitor": map[string]interface{}{
					"duration.us": isdef.IsDuration,
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
		[]validator.Validator{validatorMaker("http://foo.com"), validatorMaker("http://bar.com")},
		nil,
	})
}

func TestMultiJobConts(t *testing.T) {
	fields := fields{"myid", "myname", "mytyp"}

	uniqScope := isdef.ScopedIsUnique()

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

	contJobValidator := func(u string, msg string) validator.Validator {
		return lookslike.Compose(
			urlValidator(t, u),
			lookslike.MustCompile(map[string]interface{}{"cont": msg}),
			lookslike.MustCompile(map[string]interface{}{
				"monitor": map[string]interface{}{
					"duration.us": isdef.IsDuration,
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
		[]validator.Validator{
			contJobValidator("http://foo.com", "1st"),
			lookslike.Compose(
				contJobValidator("http://foo.com", "2nd"),
				summaryValidator(2, 0),
			),
			contJobValidator("http://bar.com", "1st"),
			lookslike.Compose(
				contJobValidator("http://bar.com", "2nd"),
				summaryValidator(2, 0),
			),
		},
		nil,
	})
}

func TestMultiJobContsCancelledEvents(t *testing.T) {
	fields := fields{"myid", "myname", "mytyp"}

	uniqScope := isdef.ScopedIsUnique()

	makeContJob := func(t *testing.T, u string) jobs.Job {
		return func(event *beat.Event) ([]jobs.Job, error) {
			eventext.MergeEventFields(event, common.MapStr{"cont": "1st"})
			eventext.CancelEvent(event)
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

	contJobValidator := func(u string, msg string) validator.Validator {
		return lookslike.Compose(
			urlValidator(t, u),
			lookslike.MustCompile(map[string]interface{}{"cont": msg}),
			lookslike.MustCompile(map[string]interface{}{
				"monitor": map[string]interface{}{
					"duration.us": isdef.IsDuration,
					"id":          uniqScope.IsUniqueTo(u),
					"name":        fields.name,
					"type":        fields.typ,
					"status":      "up",
					"check_group": uniqScope.IsUniqueTo(u),
				},
			}),
		)
	}

	metaCancelledValidator := lookslike.MustCompile(map[string]interface{}{eventext.EventCancelledMetaKey: true})
	testCommonWrap(t, testDef{
		"multi-job-continuations",
		fields,
		[]jobs.Job{makeContJob(t, "http://foo.com"), makeContJob(t, "http://bar.com")},
		[]validator.Validator{
			lookslike.Compose(
				contJobValidator("http://foo.com", "1st"),
			),
			lookslike.Compose(
				contJobValidator("http://foo.com", "2nd"),
				summaryValidator(1, 0),
			),
			lookslike.Compose(
				contJobValidator("http://bar.com", "1st"),
			),
			lookslike.Compose(
				contJobValidator("http://bar.com", "2nd"),
				summaryValidator(1, 0),
			),
		},
		[]validator.Validator{
			metaCancelledValidator,
			lookslike.MustCompile(isdef.IsEqual(common.MapStr(nil))),
			metaCancelledValidator,
			lookslike.MustCompile(isdef.IsEqual(common.MapStr(nil))),
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

func urlValidator(t *testing.T, u string) validator.Validator {
	parsed, err := url.Parse(u)
	require.NoError(t, err)
	return lookslike.MustCompile(map[string]interface{}{"url": map[string]interface{}(URLFields(parsed))})
}

// This duplicates hbtest.SummaryChecks to avoid an import cycle.
// It could be refactored out, but it just isn't worth it.
func summaryValidator(up int, down int) validator.Validator {
	return lookslike.MustCompile(map[string]interface{}{
		"summary": map[string]interface{}{
			"up":   uint16(up),
			"down": uint16(down),
		},
	})
}
