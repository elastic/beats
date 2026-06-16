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
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/reader/etw"
)

// Event Correlation System
//
// The ETW file integrity monitoring system implements event correlation
// to provide meaningful, consolidated file integrity events from raw ETW kernel events.
//
// Problem Statement:
// Raw ETW events are very granular - a simple file modification might generate:
// 1. fileNameCreate (when opening the file)
// 2. fileCreate (creating the file handle)
// 3. fileWrite (writing data)
// 4. fileSetInformation (updating metadata)
// 5. fileClose (closing the handle)
//
// These individual events are not useful for file integrity monitoring - users want
// to know "file X was modified" rather than receiving 5 separate low-level events.
//
// Correlation Strategy:
// The correlator groups related events by:
// - File object identifier (unique per file handle)
// - Process that performed the operations
// - Time proximity (events within a reasonable time window)
//
// Event Grouping:
// Events are grouped into operationGroup structures that track:
// - All related ETW events for a file handle
// - Process context (PID, user, process start key)
// - Time span of the operation group
// - Final action classification (Created, Modified, Deleted, etc.)
//
// Action Determination:
// The correlator analyzes the collection of events to determine the primary action:
// Moved > Created > Deleted > Updated > ConfigChange > AttributesModified > None
//
// Timeout and Flushing:
// Operations are considered complete when:
// - fileClose event is received (natural completion)
// - Timeout expires (configurable, default ~1 minute)
// - System shutdown (flush all pending operations)

// operationGroup represents a collection of events for a single file handle
type operationGroup struct {
	processStartKey processStartKey
	pid             uint32
	sid             string
	fileObject      fileObject
	events          []*etw.RenderedEtwEvent
	eventIDs        map[uint16]bool
	start           time.Time
	end             time.Time
	path            string
	targetPath      string
}

func (g *operationGroup) add(path string, event *etw.RenderedEtwEvent) {
	g.events = append(g.events, event)
	g.eventIDs[event.EventID] = true

	if g.start.IsZero() || event.Timestamp.Before(g.start) {
		g.start = event.Timestamp
	}
	if g.end.IsZero() || event.Timestamp.After(g.end) {
		g.end = event.Timestamp
	}

	// Capture path from first event that has it
	// for fileSetLinkPath and fileRenamePath,
	// we override it since the original open is on the source file
	if g.path == "" && path != "" {
		g.path = path
	}
	if (g.targetPath == "" ||
		event.EventID == fileSetLinkPath ||
		event.EventID == fileRenamePath) &&
		path != "" {
		g.targetPath = path
	}
	// Capture sid from first event that has it
	if g.sid == "" {
		if sid := getStringExtendedData(event, "SID"); sid != "" {
			g.sid = sid
		}
	}

	if g.pid == 0 {
		g.pid = event.ProcessID
	}
}

type actionDetector struct{}

func (ad *actionDetector) detectAction(group *operationGroup) Action {
	if group.eventIDs[fileRenamePath] {
		return Moved
	}

	if group.eventIDs[fileNameCreate] || group.eventIDs[fileSetLinkPath] {
		return Created
	}

	if group.eventIDs[fileDeletePath] {
		return Deleted
	}

	if group.eventIDs[fileSetSecurity] {
		return ConfigChange
	}

	if group.eventIDs[fileWrite] {
		return Updated
	}

	if group.eventIDs[fileSetInformation] || group.eventIDs[fileSetEA] {
		return AttributesModified
	}

	return None
}

type etwOp struct {
	fileObject      fileObject
	processStartKey processStartKey
	pid             uint32
	sid             string
	path            string
	start           time.Time
	end             time.Time
	action          Action
}

type operationsCorrelator struct {
	activeGroups map[fileObject]*operationGroup
	detector     *actionDetector
	mutex        sync.Mutex
}

func newOperationsCorrelator() *operationsCorrelator {
	return &operationsCorrelator{
		activeGroups: make(map[fileObject]*operationGroup),
		detector:     &actionDetector{},
	}
}

func (c *operationsCorrelator) findGroup(fileObject fileObject, processStartKey processStartKey, path string, event *etw.RenderedEtwEvent) (*operationGroup, bool) {
	if event.EventID != fileNameCreate {
		group, found := c.activeGroups[fileObject]
		return group, found
	}
	var group *operationGroup
	// find the most recent group with the same path
	for _, g := range c.activeGroups {
		if (g.path == path || g.targetPath == path) && g.processStartKey == processStartKey {
			if group == nil || g.start.After(group.start) {
				group = g
			}
		}
	}
	if group != nil {
		return group, true
	}
	return nil, false
}

func (c *operationsCorrelator) processEvent(path string, event *etw.RenderedEtwEvent) []*etwOp {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	fileObj := fileObject(getUint64Property(event, "FileObject"))
	if fileObj == 0 && event.EventID != fileNameCreate {
		// fileNameCreate events do not have a FileObject
		// so any other event without a FileObject is ignored
		return nil
	}
	processStartKey := processStartKey(getUint64ExtendedData(event, "PROCESS_START_KEY"))

	group, found := c.findGroup(fileObj, processStartKey, path, event)
	if !found {
		if event.EventID == fileNameCreate {
			// For orphaned fileNameCreate events, we dispatch them as
			// we will not be able to correlate them later
			return []*etwOp{{
				fileObject:      fileObj,
				processStartKey: processStartKey,
				pid:             event.ProcessID,
				sid:             getStringExtendedData(event, "SID"),
			}}
		}
		group = &operationGroup{
			fileObject:      fileObj,
			processStartKey: processStartKey,
			eventIDs:        make(map[uint16]bool),
		}
		c.activeGroups[fileObj] = group
	}

	group.add(path, event)

	if event.EventID == fileClose {
		return c.finalizeGroup(group.fileObject)
	}

	return nil
}

func (c *operationsCorrelator) finalizeGroup(fileObj fileObject) []*etwOp {
	group, exists := c.activeGroups[fileObj]
	if !exists {
		return nil
	}

	delete(c.activeGroups, fileObj)

	action := c.detector.detectAction(group)
	if action == None {
		return nil
	}

	path := group.path
	switch action {
	case Moved:
		if (group.path != "" && group.targetPath != "") &&
			group.eventIDs[fileRenamePath] &&
			group.eventIDs[fileNameCreate] {
			// Handle rename as two operations if we have both
			return c.createRenameOperations(group)
		}
	case Created:
		if group.eventIDs[fileSetLinkPath] {
			path = group.targetPath // Use target path for link creation
		}
	}

	// Normal single operation
	return []*etwOp{{
		fileObject:      group.fileObject,
		processStartKey: group.processStartKey,
		pid:             group.pid,
		sid:             group.sid,
		path:            path,
		start:           group.start,
		end:             group.end,
		action:          action,
	}}
}

// createRenameOperations creates two operations for a rename: moved (source) and created (target)
func (c *operationsCorrelator) createRenameOperations(group *operationGroup) []*etwOp {
	var operations []*etwOp
	var rename, created *etw.RenderedEtwEvent
	// the rename and created events to adjust times
	for _, event := range group.events {
		if event.EventID == fileRenamePath {
			rename = event
			continue
		}
		if event.EventID == fileNameCreate {
			created = event
		}
	}

	// Create "moved" operation for the source file
	movedOp := &etwOp{
		fileObject:      group.fileObject,
		processStartKey: group.processStartKey,
		pid:             group.pid,
		sid:             group.sid,
		path:            group.path,
		start:           group.start,
		end:             rename.Timestamp,
		action:          Moved,
	}
	operations = append(operations, movedOp)

	// Create "created" operation for the target file
	createdOp := &etwOp{
		fileObject:      group.fileObject,
		processStartKey: group.processStartKey,
		pid:             group.pid,
		sid:             group.sid,
		path:            group.targetPath,
		start:           created.Timestamp,
		end:             group.end,
		action:          Created,
	}
	operations = append(operations, createdOp)
	return operations
}

func (c *operationsCorrelator) flushExpiredGroups(timeout time.Duration) []*etwOp {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	cutoffTime := time.Now().Add(-timeout)
	var expiredOps []*etwOp
	var expiredObjects []fileObject

	for fileObject, group := range c.activeGroups {
		if group.start.Before(cutoffTime) {
			expiredObjects = append(expiredObjects, fileObject)
		}
	}

	for _, fileObject := range expiredObjects {
		expiredOps = append(expiredOps, c.finalizeGroup(fileObject)...)
	}

	return expiredOps
}
