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

import "strings"

type dKey struct {
	Ino      uint64
	DevMajor uint32
	DevMinor uint32
}

type dEntryChildren map[string]*dEntry

type dEntry struct {
	Parent   *dEntry
	Depth    uint32
	Children dEntryChildren
	Name     string
	Ino      uint64
	DevMajor uint32
	DevMinor uint32
}

func (d *dEntry) GetParent() *dEntry {
	if d == nil {
		return nil
	}

	return d.Parent
}

func pathRecursive(d *dEntry, buffer *strings.Builder, size int) {
	nameLen := len(d.Name)

	if d.Parent == nil {
		size += nameLen
		buffer.Grow(size)
		buffer.WriteString(d.Name)
		return
	}

	size += nameLen + 1
	pathRecursive(d.Parent, buffer, size)
	buffer.WriteByte('/')
	buffer.WriteString(d.Name)
}

func (d *dEntry) Path() string {
	if d == nil {
		return ""
	}

	var buffer strings.Builder
	pathRecursive(d, &buffer, 0)
	defer buffer.Reset()
	return buffer.String()
}

// releaseRecursive recursive func to satisfy the needs of Release.
func releaseRecursive(val *dEntry) {
	for _, child := range val.Children {
		releaseRecursive(child)
		delete(val.Children, child.Name)
	}

	val.Children = nil
	val.Parent = nil
}

// Release releases the resources associated with the given dEntry and all its children.
func (d *dEntry) Release() {
	if d == nil {
		return
	}

	releaseRecursive(d)
}

func (d *dEntry) RemoveChild(name string) {
	if d == nil || d.Children == nil {
		return
	}

	delete(d.Children, name)
}

// AddChild adds a child entry to the dEntry.
func (d *dEntry) AddChild(child *dEntry) {
	if d == nil || child == nil {
		return
	}

	if d.Children == nil {
		d.Children = make(map[string]*dEntry)
	}

	child.Parent = d
	child.Depth = d.Depth + 1

	d.Children[child.Name] = child
}

// GetChild returns the child entry with the given name, if it exists. Otherwise, nil is returned.
func (d *dEntry) GetChild(name string) *dEntry {
	if d == nil || d.Children == nil {
		return nil
	}

	child, exists := d.Children[name]
	if !exists {
		return nil
	}

	return child
}
