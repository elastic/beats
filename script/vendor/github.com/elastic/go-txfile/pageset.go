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

package txfile

type pageSet map[PageID]struct{}

func (s *pageSet) Add(id PageID) {
	if *s == nil {
		*s = pageSet{}
	}
	(*s)[id] = struct{}{}
}

func (s *pageSet) AddSet(other pageSet) {
	if *s == nil {
		*s = pageSet{}
	}
	for id := range other {
		(*s)[id] = struct{}{}
	}
}

func (s pageSet) Remove(id PageID) {
	if s != nil {
		delete(s, id)
	}
}

func (s pageSet) Has(id PageID) bool {
	if s != nil {
		_, exists := s[id]
		return exists
	}
	return false
}

func (s pageSet) Empty() bool { return s.Count() == 0 }

func (s pageSet) Count() int { return len(s) }

func (s pageSet) IDs() idList {
	L := len(s)
	if L == 0 {
		return nil
	}

	l, i := make(idList, L), 0
	for id := range s {
		l[i], i = id, i+1
	}
	return l
}

func (s pageSet) Regions() regionList {
	if len(s) == 0 {
		return nil
	}

	regions, i := make(regionList, len(s)), 0
	for id := range s {
		regions[i], i = region{id: id, count: 1}, i+1
	}
	optimizeRegionList(&regions)

	return regions
}
