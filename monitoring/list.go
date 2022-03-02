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

import (
	"sync"
)

// UniqueList is used to collect a list of items (strings) and get the total count and all unique strings.
type UniqueList struct {
	sync.Mutex
	list map[string]int
}

// NewUniqueList create a new UniqueList
func NewUniqueList() *UniqueList {
	return &UniqueList{
		list: map[string]int{},
	}
}

// Add adds an item to the list and increases the count for it.
func (l *UniqueList) Add(item string) {
	l.Lock()
	defer l.Unlock()
	l.list[item]++
}

// Remove removes and item for the list and decreases the count.
func (l *UniqueList) Remove(item string) {
	l.Lock()
	defer l.Unlock()
	l.list[item]--
}

// Report can be used as reporting function for monitoring.
// It reports a total count value and a names array with all the items.
func (l *UniqueList) Report(m Mode, V Visitor) {
	V.OnRegistryStart()
	defer V.OnRegistryFinished()

	var items []string
	var count int64

	l.Lock()
	defer l.Unlock()

	for key, val := range l.list {
		if val > 0 {
			items = append(items, key)
		}
		count += int64(val)
	}

	ReportInt(V, "count", count)
	ReportStringSlice(V, "names", items)
}
