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

//go:build linux

package kprobes

import (
	"context"
	"testing"

	"github.com/elastic/beats/v7/auditbeat/tracing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

type EmitterMock struct {
	mock.Mock
}

func (e *EmitterMock) Emit(ePath string, pid uint32, op uint32) error {
	args := e.Called(ePath, pid, op)
	return args.Error(0)
}

func Test_EventProcessor_process(t *testing.T) {
	type emitted struct {
		path string
		pid  uint32
		op   uint32
	}

	cases := []struct {
		name        string
		statMatches []statMatch
		events      []*ProbeEvent
		emits       []emitted
		isRecursive bool
	}{
		{
			"recursive_processor",
			[]statMatch{
				{
					ino:        1,
					major:      1,
					minor:      1,
					depth:      0,
					fileName:   "root",
					isFromMove: false,
					tid:        0,
					fullPath:   "/root/test",
				},
				{
					ino:        10,
					major:      1,
					minor:      1,
					depth:      0,
					fileName:   "root2",
					isFromMove: false,
					tid:        0,
					fullPath:   "/root2/test",
				},
			},
			[]*ProbeEvent{
				{
					// shouldn't add to cache
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskMonitor:  1,
					FileName:     "root",
					FileIno:      1,
					FileDevMajor: 100,
					FileDevMinor: 100,
				},
				{
					// shouldn't add to cache
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskMonitor:  1,
					FileName:     "root",
					FileIno:      1,
					FileDevMajor: 200,
					FileDevMinor: 200,
				},
				{
					// should add to cache but no event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskMonitor:  1,
					FileName:     "root",
					FileIno:      1,
					FileDevMajor: 1,
					FileDevMinor: 1,
				},
				{
					// should add to cache but no event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskMonitor:  1,
					FileName:     "root2",
					FileIno:      10,
					FileDevMajor: 1,
					FileDevMinor: 1,
				},
				{
					// should emit create event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskCreate:     1,
					ParentDevMinor: 1,
					ParentIno:      1,
					ParentDevMajor: 1,
					FileName:       "test_create",
					FileIno:        2,
					FileDevMajor:   1,
					FileDevMinor:   1,
				},
				{
					// should not emit create event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskCreate:     1,
					ParentDevMinor: 1,
					ParentIno:      3,
					ParentDevMajor: 1,
					FileName:       "test_create",
					FileIno:        2,
					FileDevMajor:   1,
					FileDevMinor:   1,
				},
				{
					// should not emit modify event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskModify:   1,
					FileIno:      3,
					FileDevMajor: 1,
					FileDevMinor: 1,
				},
				{
					// should emit modify event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskModify:   1,
					FileIno:      2,
					FileDevMajor: 1,
					FileDevMinor: 1,
				},
				{
					// should not emit attrib event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskAttrib:   1,
					FileIno:      3,
					FileDevMajor: 1,
					FileDevMinor: 1,
				},
				{
					// should emit attrib event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskAttrib:   1,
					FileIno:      2,
					FileDevMajor: 1,
					FileDevMinor: 1,
				},
				{
					// should emit delete event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskDelete:     1,
					ParentDevMinor: 1,
					ParentIno:      1,
					ParentDevMajor: 1,
					FileName:       "test_create",
				},
				{
					// should not emit delete event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskDelete:     1,
					ParentDevMinor: 1,
					ParentIno:      3,
					ParentDevMajor: 1,
					FileName:       "test_create",
				},
				{
					// should emit create event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskCreate:     1,
					ParentDevMinor: 1,
					ParentIno:      10,
					ParentDevMajor: 1,
					FileName:       "test_create2",
					FileIno:        11,
					FileDevMajor:   1,
					FileDevMinor:   1,
				},
				{
					// should emit create event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskCreate:     1,
					ParentDevMinor: 1,
					ParentIno:      11,
					ParentDevMajor: 1,
					FileName:       "test_child",
					FileIno:        12,
					FileDevMajor:   1,
					FileDevMinor:   1,
				},
				{
					// should emit move_from event
					Meta: tracing.Metadata{
						PID: 2,
						TID: 2,
					},
					MaskMoveFrom:   1,
					ParentDevMinor: 1,
					ParentIno:      10,
					ParentDevMajor: 1,
					FileName:       "test_create2",
				},
				{
					// should emit two move_to events
					Meta: tracing.Metadata{
						PID: 2,
						TID: 2,
					},
					MaskMoveTo:     1,
					ParentDevMinor: 1,
					ParentIno:      1,
					ParentDevMajor: 1,
					FileName:       "test_create_moved2",
				},
				{
					// should emit two move_to events
					Meta: tracing.Metadata{
						PID: 3,
						TID: 3,
					},
					MaskMoveTo:     1,
					ParentDevMinor: 1,
					ParentIno:      1,
					ParentDevMajor: 1,
					FileName:       "test_create_moved_outside",
				},
			},
			[]emitted{
				{
					path: "/root/test/test_create",
					pid:  1,
					op:   unix.IN_CREATE,
				},
				{
					path: "/root/test/test_create",
					pid:  1,
					op:   unix.IN_MODIFY,
				},
				{
					path: "/root/test/test_create",
					pid:  1,
					op:   unix.IN_ATTRIB,
				},
				{
					path: "/root/test/test_create",
					pid:  1,
					op:   unix.IN_DELETE,
				},
				{
					path: "/root2/test/test_create2",
					pid:  1,
					op:   unix.IN_CREATE,
				},
				{
					path: "/root2/test/test_create2/test_child",
					pid:  1,
					op:   unix.IN_CREATE,
				},
				{
					path: "/root2/test/test_create2",
					pid:  2,
					op:   unix.IN_MOVED_FROM,
				},
				{
					path: "/root/test/test_create_moved2",
					pid:  2,
					op:   unix.IN_MOVED_TO,
				},
				{
					path: "/root/test/test_create_moved2/test_child",
					pid:  2,
					op:   unix.IN_MOVED_TO,
				},
				{
					path: "/root/test/test_create_moved_outside",
					pid:  3,
					op:   unix.IN_MOVED_TO,
				},
			},
			true,
		},
		{
			"nonrecursive_processor",
			[]statMatch{
				{
					ino:        10,
					major:      1,
					minor:      1,
					depth:      0,
					fileName:   "target_dir",
					isFromMove: false,
					tid:        0,
					fullPath:   "/target_dir",
				},
				{
					ino:        11,
					major:      1,
					minor:      1,
					depth:      1,
					fileName:   "track_me",
					isFromMove: false,
					tid:        0,
					fullPath:   "/target_dir/track_me",
				},
				{
					ino:        100,
					major:      1,
					minor:      1,
					depth:      1,
					fileName:   "nested",
					isFromMove: false,
					tid:        0,
					fullPath:   "/target_dir/nested",
				},
				{
					ino:        1000,
					major:      1,
					minor:      1,
					depth:      2,
					fileName:   "deeper",
					isFromMove: false,
					tid:        0,
					fullPath:   "/target_dir/nested/deeper",
				},
			},
			[]*ProbeEvent{
				{
					// shouldn't add to cache
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskMonitor:  1,
					FileName:     "target_dir",
					FileIno:      1,
					FileDevMajor: 100,
					FileDevMinor: 100,
				},
				{
					// should add to cache but no event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskMonitor:  1,
					FileName:     "target_dir",
					FileIno:      10,
					FileDevMajor: 1,
					FileDevMinor: 1,
				},
				{
					// should add to cache but no event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskMonitor:  1,
					FileName:     "track_me",
					FileIno:      11,
					FileDevMajor: 1,
					FileDevMinor: 1,
				},
				{
					// should add to cache but no event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskMonitor:  1,
					FileName:     "nested",
					FileIno:      100,
					FileDevMajor: 1,
					FileDevMinor: 1,
				},
				{
					// shouldn't add to cache and no event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskMonitor:  1,
					FileName:     "deeper",
					FileIno:      1000,
					FileDevMajor: 1,
					FileDevMinor: 1,
				},
				{
					// should emit create event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskCreate:     1,
					ParentDevMinor: 1,
					ParentIno:      10,
					ParentDevMajor: 1,
					FileName:       "test_create",
					FileIno:        12,
					FileDevMajor:   1,
					FileDevMinor:   1,
				},
				{
					// should not emit create event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskCreate:     1,
					ParentDevMinor: 1,
					ParentIno:      100,
					ParentDevMajor: 1,
					FileName:       "test_create",
					FileIno:        101,
					FileDevMajor:   1,
					FileDevMinor:   1,
				},
				{
					// should not emit modify event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskModify:   1,
					FileIno:      101,
					FileDevMajor: 1,
					FileDevMinor: 1,
				},
				{
					// should emit modify event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskModify:   1,
					FileIno:      12,
					FileDevMajor: 1,
					FileDevMinor: 1,
				},
				{
					// should emit modify event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskModify:   1,
					FileIno:      11,
					FileDevMajor: 1,
					FileDevMinor: 1,
				},
				{
					// should not emit attrib event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskAttrib:   1,
					FileIno:      101,
					FileDevMajor: 1,
					FileDevMinor: 1,
				},
				{
					// should emit attrib event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskAttrib:   1,
					FileIno:      11,
					FileDevMajor: 1,
					FileDevMinor: 1,
				},
				{
					// should emit delete event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskDelete:     1,
					ParentDevMinor: 1,
					ParentIno:      10,
					ParentDevMajor: 1,
					FileName:       "test_create",
				},
				{
					// should not emit delete event
					Meta: tracing.Metadata{
						PID: 1,
						TID: 1,
					},
					MaskDelete:     1,
					ParentDevMinor: 1,
					ParentIno:      100,
					ParentDevMajor: 1,
					FileName:       "test_create",
				},
			},
			[]emitted{
				{
					path: "/target_dir/test_create",
					pid:  1,
					op:   unix.IN_CREATE,
				},
				{
					path: "/target_dir/test_create",
					pid:  1,
					op:   unix.IN_MODIFY,
				},
				{
					path: "/target_dir/track_me",
					pid:  1,
					op:   unix.IN_MODIFY,
				},
				{
					path: "/target_dir/track_me",
					pid:  1,
					op:   unix.IN_ATTRIB,
				},
				{
					path: "/target_dir/test_create",
					pid:  1,
					op:   unix.IN_DELETE,
				},
			},
			false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var emittedEvents []emitted

			mockEmitter := &EmitterMock{}
			mockEmitterCall := mockEmitter.On("Emit", mock.Anything, mock.Anything, mock.Anything)
			mockEmitterCall.Run(func(args mock.Arguments) {
				emittedEvents = append(emittedEvents, emitted{
					path: args.Get(0).(string),
					pid:  args.Get(1).(uint32),
					op:   args.Get(2).(uint32),
				})
				mockEmitterCall.ReturnArguments = []any{nil}
			})

			mockPathTraverser := &pathTraverserMock{}
			mockPathTraverserCall := mockPathTraverser.On("GetMonitorPath", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			mockPathTraverserCall.Run(func(args mock.Arguments) {
				ino := args.Get(0).(uint64)
				major := args.Get(1).(uint32)
				minor := args.Get(2).(uint32)
				name := args.Get(3).(string)
				if len(c.statMatches) == 0 {
					mockPathTraverserCall.ReturnArguments = []any{MonitorPath{}, false}
					return
				}

				if c.statMatches[0].ino != ino ||
					c.statMatches[0].major != major ||
					c.statMatches[0].minor != minor ||
					c.statMatches[0].fileName != name {
					mockPathTraverserCall.ReturnArguments = []any{MonitorPath{}, false}
					return
				}

				mockPathTraverserCall.ReturnArguments = []any{MonitorPath{
					fullPath:   c.statMatches[0].fullPath,
					depth:      c.statMatches[0].depth,
					isFromMove: c.statMatches[0].isFromMove,
					tid:        c.statMatches[0].tid,
				}, true}

				c.statMatches = c.statMatches[1:]
			})

			mockPathTraverser.On("WalkAsync", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				pid := args.Get(2).(uint32)

				c.statMatches = append(c.statMatches, statMatch{
					fullPath:   args.Get(0).(string),
					depth:      args.Get(1).(uint32),
					ino:        20,
					major:      1,
					minor:      1,
					isFromMove: true,
					fileName:   "test_create_moved_outside",
					tid:        pid,
				})

				c.events = append(c.events, []*ProbeEvent{
					{
						Meta:           tracing.Metadata{PID: 1, TID: 1},
						MaskMonitor:    1,
						ParentDevMinor: 1,
						ParentIno:      1,
						ParentDevMajor: 1,
						FileName:       "test_create_moved_outside",
						FileIno:        20,
						FileDevMajor:   1,
						FileDevMinor:   1,
					},
				}...)
			})

			eProc := newEventProcessor(mockPathTraverser, mockEmitter, c.isRecursive)
			for len(c.events) > 0 {
				err := eProc.process(context.TODO(), c.events[0])
				require.NoError(t, err)
				c.events = c.events[1:]
			}

			require.Equal(t, c.emits, emittedEvents)
		})
	}
}
