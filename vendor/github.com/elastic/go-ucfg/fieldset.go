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

package ucfg

type fieldSet struct {
	fields map[string]struct{}
	parent *fieldSet
}

func newFieldSet(parent *fieldSet) *fieldSet {
	return &fieldSet{
		fields: map[string]struct{}{},
		parent: parent,
	}
}

func (s *fieldSet) Has(name string) (exists bool) {
	if _, exists = s.fields[name]; !exists && s.parent != nil {
		exists = s.parent.Has(name)
	}
	return
}

func (s *fieldSet) Add(name string) {
	s.fields[name] = struct{}{}
}

func (s *fieldSet) AddNew(name string) (ok bool) {
	if ok = !s.Has(name); ok {
		s.Add(name)
	}
	return
}

func (s *fieldSet) Names() []string {
	var names []string
	for k := range s.fields {
		names = append(names, k)
	}

	if s.parent != nil {
		names = append(names, s.parent.Names()...)
	}
	return names
}
