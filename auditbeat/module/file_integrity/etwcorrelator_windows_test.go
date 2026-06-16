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

//go:build windows

package file_integrity

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/reader/etw"
)

// newMockEvent creates a valid RenderedEtwEvent for testing.
func newMockEvent(id uint16, fileObj fileObject, key processStartKey, pid uint32, sid, path string) *etw.RenderedEtwEvent {
	evt := &etw.RenderedEtwEvent{
		EventID:      id,
		ProcessID:    pid,
		Timestamp:    time.Now(),
		Properties:   make([]etw.RenderedProperty, 0, 1),
		ExtendedData: make([]etw.RenderedExtendedData, 0, 2),
	}

	// Add FileObject property
	prop := etw.RenderedProperty{Name: "FileObject", Value: fmt.Sprint(fileObj)}
	evt.Properties = append(evt.Properties, prop)

	prop = etw.RenderedProperty{Name: "FilePath", Value: path}
	evt.Properties = append(evt.Properties, prop)

	prop = etw.RenderedProperty{Name: "FileName", Value: path}
	evt.Properties = append(evt.Properties, prop)

	// Add PROCESS_START_KEY extended data
	extKey := etw.RenderedExtendedData{ExtType: "PROCESS_START_KEY", Data: fmt.Sprint(key)}
	evt.ExtendedData = append(evt.ExtendedData, extKey)

	// Add SID extended data
	extSid := etw.RenderedExtendedData{ExtType: "SID", Data: sid}
	evt.ExtendedData = append(evt.ExtendedData, extSid)
	return evt
}

func TestOperationsCorrelator(t *testing.T) {
	const (
		testFileObject fileObject      = 100
		testProcKey    processStartKey = 200
		testPID        uint32          = 300
		testSID        string          = "S-1-5-21"
		sourcePath     string          = "C:\\temp\\source.txt"
		targetPath     string          = "C:\\temp\\target.txt"
	)

	testCases := []struct {
		name            string
		events          []*etw.RenderedEtwEvent
		eventPaths      []string
		expectedOps     int
		expectedActions []Action
	}{
		{
			name: "Simple Create",
			events: []*etw.RenderedEtwEvent{
				newMockEvent(fileCreate, testFileObject, testProcKey, testPID, testSID, sourcePath),
				newMockEvent(fileNameCreate, 0, testProcKey, testPID, testSID, sourcePath),
				newMockEvent(fileClose, testFileObject, testProcKey, testPID, testSID, sourcePath),
			},
			eventPaths:      []string{sourcePath, sourcePath, sourcePath},
			expectedOps:     1,
			expectedActions: []Action{Created},
		},
		{
			name: "Simple Modify",
			events: []*etw.RenderedEtwEvent{
				newMockEvent(fileCreate, testFileObject, testProcKey, testPID, testSID, sourcePath),
				newMockEvent(fileWrite, testFileObject, testProcKey, testPID, testSID, sourcePath),
				newMockEvent(fileClose, testFileObject, testProcKey, testPID, testSID, sourcePath),
			},
			eventPaths:      []string{sourcePath, sourcePath, sourcePath},
			expectedOps:     1,
			expectedActions: []Action{Updated},
		},
		{
			name: "Simple Delete",
			events: []*etw.RenderedEtwEvent{
				newMockEvent(fileCreate, testFileObject, testProcKey, testPID, testSID, sourcePath),
				newMockEvent(fileDeletePath, testFileObject, testProcKey, testPID, testSID, sourcePath),
				newMockEvent(fileClose, testFileObject, testProcKey, testPID, testSID, sourcePath),
			},
			eventPaths:      []string{sourcePath, sourcePath, sourcePath},
			expectedOps:     1,
			expectedActions: []Action{Deleted},
		},
		{
			name: "Complex Rename - Moved and Created",
			events: []*etw.RenderedEtwEvent{
				newMockEvent(fileCreate, testFileObject, testProcKey, testPID, testSID, sourcePath),
				newMockEvent(fileWrite, testFileObject, testProcKey, testPID, testSID, sourcePath),
				newMockEvent(fileRenamePath, testFileObject, testProcKey, testPID, testSID, sourcePath),
				newMockEvent(fileNameCreate, 0, testProcKey, testPID, testSID, targetPath),
				newMockEvent(fileClose, testFileObject, testProcKey, testPID, testSID, sourcePath),
			},
			eventPaths:      []string{sourcePath, sourcePath, targetPath, targetPath, targetPath},
			expectedOps:     2,
			expectedActions: []Action{Moved, Created},
		},
		{
			name:            "Orphaned fileNameCreate",
			events:          []*etw.RenderedEtwEvent{newMockEvent(fileNameCreate, 0, testProcKey, testPID, testSID, sourcePath)},
			eventPaths:      []string{sourcePath},
			expectedOps:     1,
			expectedActions: nil, // Orphaned events don't have a final action
		},
		{
			name: "No Action on Close",
			events: []*etw.RenderedEtwEvent{
				newMockEvent(fileCreate, testFileObject, testProcKey, testPID, testSID, sourcePath),
				newMockEvent(fileClose, testFileObject, testProcKey, testPID, testSID, sourcePath),
			},
			eventPaths:      []string{sourcePath, sourcePath},
			expectedOps:     0,
			expectedActions: []Action{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			correlator := newOperationsCorrelator()
			var finalOps []*etwOp

			for i, event := range tc.events {
				ops := correlator.processEvent(tc.eventPaths[i], event)
				finalOps = append(finalOps, ops...)
			}

			if len(finalOps) != tc.expectedOps {
				t.Fatalf("Expected %d final operations, but got %d", tc.expectedOps, len(finalOps))
			}

			if tc.expectedActions != nil {
				var gotActions []Action
				for _, op := range finalOps {
					gotActions = append(gotActions, op.action)
				}
				actionMatch := reflect.DeepEqual(actionsToSet(gotActions), actionsToSet(tc.expectedActions))
				if !actionMatch {
					t.Errorf("Expected actions %v, but got %v", tc.expectedActions, gotActions)
				}
			}
		})
	}

	t.Run("Timeout Flush", func(t *testing.T) {
		correlator := newOperationsCorrelator()
		timeout := 10 * time.Millisecond

		correlator.processEvent(sourcePath, newMockEvent(fileWrite, testFileObject, testProcKey, testPID, testSID, sourcePath))
		time.Sleep(timeout + 5*time.Millisecond)
		ops := correlator.flushExpiredGroups(timeout)

		if len(ops) != 1 {
			t.Fatalf("Expected 1 flushed operation, but got %d", len(ops))
		}
		if ops[0].action != Updated {
			t.Errorf("Expected flushed action to be Updated, but got %v", ops[0].action)
		}
	})
}

// Helper to convert a slice of actions to a map (set) for order-insensitive comparison.
func actionsToSet(actions []Action) map[Action]bool {
	set := make(map[Action]bool)
	for _, a := range actions {
		set[a] = true
	}
	return set
}
