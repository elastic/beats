// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBatchCursorTime(t *testing.T) {
	tests := map[string]struct {
		raw     string
		want    time.Time
		wantErr string
	}{
		"batch cursor layout": {
			raw:  "2024-01-01T11:55:00.000Z",
			want: time.Date(2024, time.January, 1, 11, 55, 0, 0, time.UTC),
		},
		"salesforce event layout": {
			raw:  "2024-01-01T11:55:00.000+0000",
			want: time.Date(2024, time.January, 1, 11, 55, 0, 0, time.UTC),
		},
		"rfc3339 layout": {
			raw:  "2024-01-01T11:55:00Z",
			want: time.Date(2024, time.January, 1, 11, 55, 0, 0, time.UTC),
		},
		"rfc3339 nano layout": {
			raw:  "2024-01-01T11:55:00.123456789Z",
			want: time.Date(2024, time.January, 1, 11, 55, 0, 123456789, time.UTC),
		},
		"custom rfc3339 like layout": {
			raw:  "2024-01-01T11:55:00.123Z",
			want: time.Date(2024, time.January, 1, 11, 55, 0, 123000000, time.UTC),
		},
		"invalid layout": {
			raw:     "not-a-time",
			wantErr: `unsupported Salesforce cursor time format: "not-a-time"`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := parseBatchCursorTime(tc.raw)
			if tc.wantErr != "" {
				require.Error(t, err, "expected invalid batch cursor time to fail")
				assert.EqualError(t, err, tc.wantErr)
				return
			}

			require.NoError(t, err, "expected supported batch cursor time layout to parse")
			assert.True(t, got.Equal(tc.want), "expected parsed batch cursor time to match the canonical UTC instant")
		})
	}
}

func TestNextObjectBatchWindow(t *testing.T) {
	runEnd := time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		cursor  dateTimeCursor
		want    objectBatchWindow
		wantOK  bool
		wantErr string
	}{
		"first run uses initial interval": {
			cursor: dateTimeCursor{},
			want: objectBatchWindow{
				Start: time.Date(2024, time.January, 1, 11, 45, 0, 0, time.UTC),
				End:   time.Date(2024, time.January, 1, 11, 50, 0, 0, time.UTC),
			},
			wantOK: true,
		},
		"resume uses persisted progress time": {
			cursor: dateTimeCursor{
				ProgressTime: "2024-01-01T11:55:00.000Z",
			},
			want: objectBatchWindow{
				Start: time.Date(2024, time.January, 1, 11, 55, 0, 0, time.UTC),
				End:   time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC),
			},
			wantOK: true,
		},
		"batch end clamps to run end": {
			cursor: dateTimeCursor{
				ProgressTime: "2024-01-01T11:58:00.000Z",
			},
			want: objectBatchWindow{
				Start: time.Date(2024, time.January, 1, 11, 58, 0, 0, time.UTC),
				End:   time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC),
			},
			wantOK: true,
		},
		"caught up returns no window": {
			cursor: dateTimeCursor{
				ProgressTime: "2024-01-01T12:00:00.000Z",
			},
			wantOK: false,
		},
		"invalid progress time fails": {
			cursor: dateTimeCursor{
				ProgressTime: "not-a-time",
			},
			wantErr: `unsupported Salesforce cursor time format: "not-a-time"`,
		},
		"legacy first_event_time seeds upgrade window": {
			cursor: dateTimeCursor{
				FirstEventTime: "2024-01-01T11:55:00.000+0000",
				LastEventTime:  "2024-01-01T11:54:30.000+0000",
			},
			want: objectBatchWindow{
				Start: time.Date(2024, time.January, 1, 11, 55, 0, 0, time.UTC),
				End:   time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC),
			},
			wantOK: true,
		},
		"legacy last_event_time only seeds upgrade window": {
			cursor: dateTimeCursor{
				LastEventTime: "2024-01-01T11:57:00.000+0000",
			},
			want: objectBatchWindow{
				Start: time.Date(2024, time.January, 1, 11, 57, 0, 0, time.UTC),
				End:   time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC),
			},
			wantOK: true,
		},
		"progress_time takes precedence over legacy watermarks": {
			cursor: dateTimeCursor{
				ProgressTime:   "2024-01-01T11:58:00.000Z",
				FirstEventTime: "2024-01-01T10:00:00.000+0000",
				LastEventTime:  "2024-01-01T09:00:00.000+0000",
			},
			want: objectBatchWindow{
				Start: time.Date(2024, time.January, 1, 11, 58, 0, 0, time.UTC),
				End:   time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC),
			},
			wantOK: true,
		},
		"newer first_event_time projects progress_time forward on re-enable": {
			cursor: dateTimeCursor{
				ProgressTime:   "2024-01-01T11:50:00.000Z",
				FirstEventTime: "2024-01-01T11:59:25.438+0000",
				LastEventTime:  "2024-01-01T11:59:25.438+0000",
			},
			want: objectBatchWindow{
				Start: time.Date(2024, time.January, 1, 11, 59, 25, 438000000, time.UTC),
				End:   time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC),
			},
			wantOK: true,
		},
		"newer last_event_time projects progress_time forward on re-enable": {
			cursor: dateTimeCursor{
				ProgressTime:  "2024-01-01T11:50:00.000Z",
				LastEventTime: "2024-01-01T11:57:30.000+0000",
			},
			want: objectBatchWindow{
				Start: time.Date(2024, time.January, 1, 11, 57, 30, 0, time.UTC),
				End:   time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC),
			},
			wantOK: true,
		},
		"unparseable legacy watermark leaves progress_time start unchanged": {
			cursor: dateTimeCursor{
				ProgressTime:   "2024-01-01T11:55:00.000Z",
				FirstEventTime: "not-a-time",
			},
			want: objectBatchWindow{
				Start: time.Date(2024, time.January, 1, 11, 55, 0, 0, time.UTC),
				End:   time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC),
			},
			wantOK: true,
		},
		"invalid legacy first_event_time fails loud": {
			cursor: dateTimeCursor{
				FirstEventTime: "not-a-time",
			},
			wantErr: `unsupported Salesforce cursor time format: "not-a-time"`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s := &salesforceInput{
				cursor: &state{Object: tc.cursor},
				srcConfig: &config{
					EventMonitoringMethod: &eventMonitoringMethod{
						Object: EventMonitoringConfig{
							Batch: &batchConfig{
								Enabled:          pointer(true),
								InitialInterval:  15 * time.Minute,
								MaxWindowsPerRun: pointer(1),
								Window:           5 * time.Minute,
							},
						},
					},
				},
			}

			got, ok, err := s.nextObjectBatchWindow(runEnd)
			if tc.wantErr != "" {
				require.Error(t, err, "expected invalid progress time to fail")
				assert.EqualError(t, err, tc.wantErr)
				return
			}

			require.NoError(t, err, "expected next batch window calculation to succeed")
			assert.Equal(t, tc.wantOK, ok, "expected next batch window availability to match")
			if tc.wantOK {
				assert.Equal(t, tc.want, got, "expected next batch window bounds to match")
			}
		})
	}
}
