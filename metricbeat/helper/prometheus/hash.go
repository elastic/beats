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

package prometheus

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"sync"

	"github.com/cespare/xxhash"

	"github.com/elastic/beats/libbeat/common"
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
func LabelHash(labelMap common.MapStr) string {
	ls := flatten("", labelMap, make(labels, 0))
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

func flatten(prefix string, in common.MapStr, out labels) labels {
	for k, v := range in {
		var fullKey string
		if prefix == "" {
			fullKey = k
		} else {
			fullKey = fmt.Sprintf("%s.%s", prefix, k)
		}

		if m, ok := tryToMapStr(v); ok {
			flatten(fullKey, m, out)
		} else {
			if val, ok := v.(string); ok {
				out = append(out, label{fullKey, val})
			}

		}
	}

	sort.Sort(out)
	return out
}

func tryToMapStr(v interface{}) (common.MapStr, bool) {
	switch m := v.(type) {
	case common.MapStr:
		return m, true
	case map[string]interface{}:
		return common.MapStr(m), true
	default:
		return nil, false
	}
}
