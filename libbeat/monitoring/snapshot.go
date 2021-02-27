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

package monitoring

import "strings"

// FlatSnapshot represents a flatten snapshot of all metrics.
// Names in the tree will be joined with `.` .
type FlatSnapshot struct {
	Bools        map[string]bool
	Ints         map[string]int64
	Floats       map[string]float64
	Strings      map[string]string
	StringSlices map[string][]string
}

type flatSnapshotVisitor struct {
	snapshot FlatSnapshot
	level    []string
}

type structSnapshotVisitor struct {
	key   keyStack
	event eventStack
	depth int
}

type keyStack struct {
	current string
	stack   []string
	stack0  [32]string
}

type eventStack struct {
	current map[string]interface{}
	stack   []map[string]interface{}
	stack0  [32]map[string]interface{}
}

// CollectFlatSnapshot collects a flattened snapshot of
// a metrics tree start with the given registry.
func CollectFlatSnapshot(r *Registry, mode Mode, expvar bool) FlatSnapshot {
	if r == nil {
		r = Default
	}

	vs := newFlatSnapshotVisitor()
	r.Visit(mode, vs)
	if expvar {
		VisitExpvars(vs)
	}
	return vs.snapshot
}

func MakeFlatSnapshot() FlatSnapshot {
	return FlatSnapshot{
		Bools:        map[string]bool{},
		Ints:         map[string]int64{},
		Floats:       map[string]float64{},
		Strings:      map[string]string{},
		StringSlices: map[string][]string{},
	}
}

// CollectStructSnapshot collects a structured metrics snaphot of
// a metrics tree starting with the given registry.
// Empty namespaces will be omitted.
func CollectStructSnapshot(r *Registry, mode Mode, expvar bool) map[string]interface{} {
	if r == nil {
		r = Default
	}

	vs := newStructSnapshotVisitor()
	r.Visit(mode, vs)
	snapshot := vs.event.current

	if expvar {
		vs := newStructSnapshotVisitor()
		VisitExpvars(vs)
		for k, v := range vs.event.current {
			snapshot[k] = v
		}
	}

	return snapshot
}

func newFlatSnapshotVisitor() *flatSnapshotVisitor {
	return &flatSnapshotVisitor{snapshot: MakeFlatSnapshot()}
}

func (vs *flatSnapshotVisitor) OnRegistryStart() {}

func (vs *flatSnapshotVisitor) OnRegistryFinished() {
	if len(vs.level) > 0 {
		vs.dropName()
	}
}

func (vs *flatSnapshotVisitor) OnKey(name string) {
	vs.level = append(vs.level, name)
}

func (vs *flatSnapshotVisitor) getName() string {
	defer vs.dropName()
	if len(vs.level) == 1 {
		return vs.level[0]
	}
	return strings.Join(vs.level, ".")
}

func (vs *flatSnapshotVisitor) dropName() {
	vs.level = vs.level[:len(vs.level)-1]
}

func (vs *flatSnapshotVisitor) OnString(s string) {
	vs.snapshot.Strings[vs.getName()] = s
}

func (vs *flatSnapshotVisitor) OnBool(b bool) {
	vs.snapshot.Bools[vs.getName()] = b
}

func (vs *flatSnapshotVisitor) OnInt(i int64) {
	vs.snapshot.Ints[vs.getName()] = i
}

func (vs *flatSnapshotVisitor) OnFloat(f float64) {
	vs.snapshot.Floats[vs.getName()] = f
}

func (vs *flatSnapshotVisitor) OnStringSlice(f []string) {
	vs.snapshot.StringSlices[vs.getName()] = f
}

func newStructSnapshotVisitor() *structSnapshotVisitor {
	vs := &structSnapshotVisitor{}
	vs.key.stack = vs.key.stack0[:0]
	vs.event.stack = vs.event.stack0[:0]
	return vs
}

func (s *structSnapshotVisitor) OnRegistryStart() {
	if s.depth > 0 {
		s.event.push()
	}
	s.depth++
}

func (s *structSnapshotVisitor) OnRegistryFinished() {
	s.depth--
	if s.depth == 0 {
		return
	}

	event := s.event.pop()
	if event == nil {
		s.key.pop()
		return
	}

	s.setValue(event)
}

func (s *structSnapshotVisitor) OnKey(key string) {
	s.key.push(key)
}

func (s *structSnapshotVisitor) OnString(str string) { s.setValue(str) }
func (s *structSnapshotVisitor) OnBool(b bool)       { s.setValue(b) }
func (s *structSnapshotVisitor) OnInt(i int64)       { s.setValue(i) }
func (s *structSnapshotVisitor) OnFloat(f float64)   { s.setValue(f) }
func (s *structSnapshotVisitor) OnStringSlice(f []string) {
	c := make([]string, len(f))
	copy(c, f)
	s.setValue(c)
}

func (s *structSnapshotVisitor) setValue(v interface{}) {
	if s.event.current == nil {
		s.event.current = map[string]interface{}{}
	}

	s.event.current[s.key.current] = v
	s.key.pop()
}

func (s *keyStack) push(key string) {
	s.stack = append(s.stack, s.current)
	s.current = key
}

func (s *keyStack) pop() {
	last := len(s.stack) - 1
	s.current = s.stack[last]
	s.stack = s.stack[:last]
}

func (s *eventStack) push() {
	s.stack = append(s.stack, s.current)
	s.current = nil
}

func (s *eventStack) pop() map[string]interface{} {
	event := s.current
	last := len(s.stack) - 1
	s.current = s.stack[last]
	s.stack = s.stack[:last]
	return event
}
