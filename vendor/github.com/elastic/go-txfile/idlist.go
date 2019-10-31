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

import "sort"

type idList []PageID

func (l *idList) Add(id PageID) {
	*l = append(*l, id)
}

func (l idList) ToSet() pageSet {
	L := len(l)
	if L == 0 {
		return nil
	}

	s := make(pageSet, L)
	for _, id := range l {
		s.Add(id)
	}
	return s
}

func (l idList) Sort() {
	sort.Slice(l, func(i, j int) bool {
		return l[i] < l[j]
	})
}

func (l idList) Regions() regionList {
	if len(l) == 0 {
		return nil
	}

	regions := make(regionList, len(l))
	for i, id := range l {
		regions[i] = region{id: id, count: 1}
	}
	optimizeRegionList(&regions)
	return regions
}
