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
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/isdef"
	"github.com/elastic/go-lookslike/testslike"
	"github.com/elastic/go-lookslike/validator"

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/hbtestllext"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/logger"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/summarizer"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/summarizer/summarizertesthelper"
	"github.com/elastic/beats/v7/heartbeat/scheduler/schedule"
	"github.com/elastic/beats/v7/libbeat/beat"
)

type testDef struct {
	name         string
	sFields      stdfields.StdMonitorFields
	jobs         []jobs.Job
	want         []validator.Validator
	metaWant     []validator.Validator
	logValidator func(t *testing.T, results []*beat.Event, observed []observer.LoggedEntry)
}

var testMonFields = stdfields.StdMonitorFields{
	ID:          "myid",
	Name:        "myname",
	Type:        "mytype",
	Schedule:    schedule.MustParse("@every 1s"),
	Timeout:     1,
	MaxAttempts: 1,
}

var testBrowserMonFields = stdfields.StdMonitorFields{
	Type:     "browser",
	Schedule: schedule.MustParse("@every 1s"),
	Timeout:  1,
}

func testCommonWrap(t *testing.T, tt testDef) {
	t.Helper()
	t.Run(tt.name, func(t *testing.T) {
		wrapped := WrapCommon(tt.jobs, tt.sFields, nil)

		core, observedLogs := observer.New(zapcore.InfoLevel)
		logger.SetLogger(logp.NewLogger("t", zap.WrapCore(func(in zapcore.Core) zapcore.Core {
			return zapcore.NewTee(in, core)
		})))

		results, err := jobs.ExecJobsAndConts(t, wrapped)
		assert.NoError(t, err)

		assert.Len(t, results, len(tt.want), "Expected test def wants to correspond exactly to number results.")
		for idx, r := range results {
			t.Run(fmt.Sprintf("result at index %d", idx), func(t *testing.T) {
				want := tt.want[idx]
				testslike.Test(t, want, r.Fields)

				if tt.metaWant != nil {
					metaWant := tt.metaWant[idx]
					testslike.Test(t, metaWant, r.Meta)
				}

			})
		}

		if tt.logValidator != nil {
			tt.logValidator(t, results, observedLogs.All())
		}
	})
}

func TestSimpleJob(t *testing.T) {
	testCommonWrap(t, testDef{
		"simple",
		testMonFields,
		[]jobs.Job{makeURLJob(t, "tcp://foo.com:80")},
		[]validator.Validator{
			lookslike.Compose(
				urlValidator(t, "tcp://foo.com:80"),
				lookslike.MustCompile(map[string]interface{}{
					"monitor": map[string]interface{}{
						"duration.us": hbtestllext.IsInt64,
						"id":          testMonFields.ID,
						"name":        testMonFields.Name,
						"type":        testMonFields.Type,
						"status":      "up",
						"check_group": isdef.IsString,
					},
				}),
				hbtestllext.MonitorTimespanValidator,
				stateValidator(),
				summarizertesthelper.SummaryValidator(1, 0),
			)},
		nil,
		func(t *testing.T, results []*beat.Event, observed []observer.LoggedEntry) {
			require.Len(t, observed, 1)
			require.Equal(t, "Monitor finished", observed[0].Message)

			durationUs, err := results[0].Fields.GetValue("monitor.duration.us")
			require.NoError(t, err)

			expectedMonitor := logger.MonitorRunInfo{
				MonitorID: testMonFields.ID,
				Type:      testMonFields.Type,
				Duration:  durationUs.(int64),
				Status:    "up",
			}
			require.ElementsMatch(t, []zap.Field{
				logp.Any("event", map[string]string{"action": logger.ActionMonitorRun}),
				logp.Any("monitor", &expectedMonitor),
			}, observed[0].Context)
		},
	})
}

func TestAdditionalStdFields(t *testing.T) {
	scenarios := []struct {
		name           string
		fieldGenerator func() stdfields.StdMonitorFields
		validator      validator.Validator
	}{
		{
			"with service name",
			func() stdfields.StdMonitorFields {
				f := testMonFields
				f.Service.Name = "mysvc"
				return f
			},
			lookslike.MustCompile(map[string]interface{}{
				"service": map[string]interface{}{
					"name": "mysvc",
				},
			}),
		},
		{
			"with origin",
			func() stdfields.StdMonitorFields {
				f := testMonFields
				f.Origin = "ui"
				return f
			},
			lookslike.MustCompile(map[string]interface{}{
				"monitor": map[string]interface{}{"origin": "ui"},
			}),
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			testCommonWrap(t, testDef{
				"simple",
				tt.fieldGenerator(),
				[]jobs.Job{makeURLJob(t, "tcp://foo.com:80")},
				[]validator.Validator{
					lookslike.Compose(
						tt.validator,
						urlValidator(t, "tcp://foo.com:80"),
						lookslike.MustCompile(map[string]interface{}{
							"monitor": map[string]interface{}{
								"duration.us": hbtestllext.IsInt64,
								"id":          testMonFields.ID,
								"name":        testMonFields.Name,
								"type":        testMonFields.Type,
								"status":      "up",
								"check_group": isdef.IsString,
							},
						}),
						stateValidator(),
						hbtestllext.MonitorTimespanValidator,
						summarizertesthelper.SummaryValidator(1, 0),
					)},
				nil,
				nil,
			})
		})
	}

}

func TestErrorJob(t *testing.T) {
	errorJob := func(event *beat.Event) ([]jobs.Job, error) {
		return nil, fmt.Errorf("myerror")
	}

	errorJobValidator := lookslike.Compose(
		stateValidator(),
		lookslike.MustCompile(map[string]interface{}{"error": map[string]interface{}{"message": "myerror", "type": "io"}}),
		lookslike.MustCompile(map[string]interface{}{
			"monitor": map[string]interface{}{
				"duration.us": hbtestllext.IsInt64,
				"id":          testMonFields.ID,
				"name":        testMonFields.Name,
				"type":        testMonFields.Type,
				"status":      "down",
				"check_group": isdef.IsString,
			},
		}),
	)

	testCommonWrap(t, testDef{
		"job error",
		testMonFields,
		[]jobs.Job{errorJob},
		[]validator.Validator{
			lookslike.Compose(
				errorJobValidator,
				hbtestllext.MonitorTimespanValidator,
				summarizertesthelper.SummaryValidator(0, 1),
			)},
		nil,
		nil,
	})
}

func TestMultiJobNoConts(t *testing.T) {
	uniqScope := isdef.ScopedIsUnique()

	validatorMaker := func(u string) validator.Validator {
		return lookslike.Compose(
			urlValidator(t, u),
			lookslike.MustCompile(map[string]interface{}{
				"monitor": map[string]interface{}{
					"duration.us": hbtestllext.IsInt64,
					"id":          uniqScope.IsUniqueTo("id"),
					"name":        testMonFields.Name,
					"type":        testMonFields.Type,
					"status":      "up",
					"check_group": uniqScope.IsUniqueTo("check_group"),
				},
			}),
			stateValidator(),
			hbtestllext.MonitorTimespanValidator,
			summarizertesthelper.SummaryValidator(1, 0),
		)
	}

	testCommonWrap(t, testDef{
		"multi-job",
		testMonFields,
		[]jobs.Job{makeURLJob(t, "http://foo.com"), makeURLJob(t, "http://bar.com")},
		[]validator.Validator{validatorMaker("http://foo.com"), validatorMaker("http://bar.com")},
		nil,
		nil,
	})
}

func TestMultiJobConts(t *testing.T) {
	uniqScope := isdef.ScopedIsUnique()

	makeContJob := func(t *testing.T, u string) jobs.Job {
		return func(event *beat.Event) ([]jobs.Job, error) {
			eventext.MergeEventFields(event, mapstr.M{"cont": "1st"})
			u, err := url.Parse(u)
			require.NoError(t, err)
			eventext.MergeEventFields(event, mapstr.M{"url": URLFields(u)})
			return []jobs.Job{
				func(event *beat.Event) ([]jobs.Job, error) {
					eventext.MergeEventFields(event, mapstr.M{"cont": "2nd"})
					eventext.MergeEventFields(event, mapstr.M{"url": URLFields(u)})
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
					"id":          uniqScope.IsUniqueTo(u),
					"name":        testMonFields.Name,
					"type":        testMonFields.Type,
					"status":      "up",
					"check_group": uniqScope.IsUniqueTo(u),
				},
				"state": isdef.Optional(hbtestllext.IsMonitorState),
			}),
			hbtestllext.MonitorTimespanValidator,
		)
	}

	testCommonWrap(t, testDef{
		"multi-job-continuations",
		testMonFields,
		[]jobs.Job{
			makeContJob(t, "http://foo.com"),
			makeContJob(t, "http://bar.com"),
		},
		[]validator.Validator{
			contJobValidator("http://foo.com", "1st"),
			lookslike.Compose(
				contJobValidator("http://foo.com", "2nd"),
				summarizertesthelper.SummaryValidator(2, 0),
			),
			contJobValidator("http://bar.com", "1st"),
			lookslike.Compose(
				contJobValidator("http://bar.com", "2nd"),
				summarizertesthelper.SummaryValidator(2, 0),
			),
		},
		nil,
		nil,
	})
}

func TestRetryMultiCont(t *testing.T) {
	uniqScope := isdef.ScopedIsUnique()

	expected := []struct {
		monStatus string
		js        summarizer.JobSummary
		state     monitorstate.State
	}{
		{
			"down",
			summarizer.JobSummary{
				Status:       "down",
				FinalAttempt: true,
				// we expect two up since this is a lightweight
				// job and all events get a monitor status
				// since no errors are returned that's 2
				Up:          0,
				Down:        2,
				Attempt:     1,
				MaxAttempts: 2,
			},
			monitorstate.State{
				Status: "down",
				Up:     0,
				Down:   2,
				Checks: 2,
			},
		},
		{
			"down",
			summarizer.JobSummary{
				Status:       "down",
				FinalAttempt: true,
				Up:           0,
				Down:         2,
				Attempt:      2,
				MaxAttempts:  2,
			},
			monitorstate.State{
				Status: "down",
				Up:     0,
				Down:   2,
				Checks: 2,
			},
		},
	}

	jobErr := fmt.Errorf("down")

	makeContJob := func(t *testing.T, u string) jobs.Job {
		expIdx := 0
		return func(event *beat.Event) ([]jobs.Job, error) {
			eventext.MergeEventFields(event, mapstr.M{"cont": "1st"})
			u, err := url.Parse(u)
			require.NoError(t, err)
			eventext.MergeEventFields(event, mapstr.M{"url": URLFields(u)})

			return []jobs.Job{
				func(event *beat.Event) ([]jobs.Job, error) {
					eventext.MergeEventFields(event, mapstr.M{"cont": "2nd"})
					eventext.MergeEventFields(event, mapstr.M{"url": URLFields(u)})

					expIdx++
					if expIdx >= len(expected)-1 {
						expIdx = 0
					}
					exp := expected[expIdx]
					if exp.js.Status == "down" {
						return nil, jobErr
					}

					return nil, nil
				},
			}, jobErr
		}
	}

	contJobValidator := func(u string, msg string) validator.Validator {
		return lookslike.Compose(
			urlValidator(t, u),
			lookslike.MustCompile(map[string]interface{}{"cont": msg}),
			lookslike.MustCompile(map[string]interface{}{
				"error": map[string]interface{}{
					"message": isdef.IsString,
					"type":    isdef.IsString,
				},
				"monitor": map[string]interface{}{
					"id":          uniqScope.IsUniqueTo(u),
					"name":        testMonFields.Name,
					"type":        testMonFields.Type,
					"status":      "down",
					"check_group": uniqScope.IsUniqueTo(u),
				},
				"state": isdef.Optional(hbtestllext.IsMonitorState),
			}),
			hbtestllext.MonitorTimespanValidator,
		)
	}

	retryMonFields := testMonFields
	retryMonFields.MaxAttempts = 2

	for _, expected := range expected {
		testCommonWrap(t, testDef{
			"multi-job-continuations-retry",
			retryMonFields,
			[]jobs.Job{makeContJob(t, "http://foo.com")},
			[]validator.Validator{
				contJobValidator("http://foo.com", "1st"),
				lookslike.Compose(
					contJobValidator("http://foo.com", "2nd"),
					summarizertesthelper.SummaryValidator(expected.js.Up, expected.js.Down),
				),
				contJobValidator("http://foo.com", "1st"),
				lookslike.Compose(
					contJobValidator("http://foo.com", "2nd"),
					summarizertesthelper.SummaryValidator(expected.js.Up, expected.js.Down),
				),
			},
			nil,
			nil,
		})
	}
}

func TestMultiJobContsCancelledEvents(t *testing.T) {
	uniqScope := isdef.ScopedIsUnique()

	makeContJob := func(t *testing.T, u string) jobs.Job {
		return func(event *beat.Event) ([]jobs.Job, error) {
			eventext.MergeEventFields(event, mapstr.M{"cont": "1st"})
			eventext.CancelEvent(event)
			u, err := url.Parse(u)
			require.NoError(t, err)
			eventext.MergeEventFields(event, mapstr.M{"url": URLFields(u)})
			return []jobs.Job{
				func(event *beat.Event) ([]jobs.Job, error) {
					eventext.MergeEventFields(event, mapstr.M{"cont": "2nd"})
					eventext.MergeEventFields(event, mapstr.M{"url": URLFields(u)})
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
					"id":          uniqScope.IsUniqueTo(u),
					"name":        testMonFields.Name,
					"type":        testMonFields.Type,
					"status":      "up",
					"check_group": uniqScope.IsUniqueTo(u),
				},
				"state": isdef.Optional(hbtestllext.IsMonitorState),
			}),
			hbtestllext.MonitorTimespanValidator,
		)
	}

	metaCancelledValidator := lookslike.MustCompile(map[string]interface{}{eventext.EventCancelledMetaKey: true})
	testCommonWrap(t, testDef{
		"multi-job-continuations",
		testMonFields,
		[]jobs.Job{makeContJob(t, "http://foo.com"), makeContJob(t, "http://bar.com")},
		[]validator.Validator{
			lookslike.Compose(
				contJobValidator("http://foo.com", "1st"),
			),
			lookslike.Compose(
				contJobValidator("http://foo.com", "2nd"),
				summarizertesthelper.SummaryValidator(1, 0),
			),
			lookslike.Compose(
				contJobValidator("http://bar.com", "1st"),
			),
			lookslike.Compose(
				contJobValidator("http://bar.com", "2nd"),
				summarizertesthelper.SummaryValidator(1, 0),
			),
		},
		[]validator.Validator{
			metaCancelledValidator,
			lookslike.MustCompile(isdef.IsEqual(mapstr.M(nil))),
			metaCancelledValidator,
			lookslike.MustCompile(isdef.IsEqual(mapstr.M(nil))),
		},
		nil,
	})
}

func makeURLJob(t *testing.T, u string) jobs.Job {
	parsed, err := url.Parse(u)
	require.NoError(t, err)
	return func(event *beat.Event) (i []jobs.Job, e error) {
		eventext.MergeEventFields(event, mapstr.M{"url": URLFields(parsed)})
		return nil, nil
	}
}

func urlValidator(t *testing.T, u string) validator.Validator {
	parsed, err := url.Parse(u)
	require.NoError(t, err)
	return lookslike.MustCompile(map[string]interface{}{"url": map[string]interface{}(URLFields(parsed))})
}

func stateValidator() validator.Validator {
	return lookslike.MustCompile(map[string]interface{}{
		"state": hbtestllext.IsMonitorState,
	})
}

func TestTimespan(t *testing.T) {
	now := time.Now()
	sched10s, err := schedule.Parse("@every 10s")
	require.NoError(t, err)

	type args struct {
		started time.Time
		sched   *schedule.Schedule
		timeout time.Duration
	}
	tests := []struct {
		name string
		args args
		want mapstr.M
	}{
		{
			"interval longer than timeout",
			args{now, sched10s, time.Second},
			mapstr.M{
				"gte": now,
				"lt":  now.Add(time.Second * 10),
			},
		},
		{
			"timeout longer than interval",
			args{now, sched10s, time.Second * 20},
			mapstr.M{
				"gte": now,
				"lt":  now.Add(time.Second * 20),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := timespan(tt.args.started, tt.args.sched, tt.args.timeout); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("timespan() = %v, want %v", got, tt.want)
			}
		})
	}
}
