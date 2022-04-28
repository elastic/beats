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

package labelhash

import (
	"bytes"
	"sort"
	"strconv"
	"sync"

	"github.com/cespare/xxhash/v2"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

const sep = '\xff'

var byteBuffer = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(nil)
	},
}

type label struct {
	key   string
	value string
}

type labels []label

func (ls labels) Len() int           { return len(ls) }
func (ls labels) Swap(i, j int)      { ls[i], ls[j] = ls[j], ls[i] }
func (ls labels) Less(i, j int) bool { return ls[i].key < ls[j].key }

// LabelHash hashes the labels map and returns a string
func LabelHash(labelMap mapstr.M) string {
	ls := make(labels, len(labelMap))

	for k, v := range labelMap {
		if val, ok := v.(string); ok {
			ls = append(ls, label{k, val})
		}
	}

	sort.Sort(ls)
	b := byteBuffer.Get().(*bytes.Buffer)
	b.Reset()

	for _, label := range ls {
		b.WriteString(label.key)
		b.WriteByte(sep)
		b.WriteString(label.value)
		b.WriteByte(sep)
	}
	hash := xxhash.Sum64(b.Bytes())
	byteBuffer.Put(b)
	return strconv.FormatUint(hash, 10)
}
