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

package dtfmt

import (
	"math"
	"strconv"
)

func appendUnpadded(bs []byte, i int) []byte {
	return strconv.AppendInt(bs, int64(i), 10)
}

func appendPadded(bs []byte, i, sz int) []byte {
	if i < 0 {
		bs = append(bs, '-')
		i = -i
	}

	if i < 10 {
		for ; sz > 1; sz-- {
			bs = append(bs, '0')
		}
		return append(bs, byte(i)+'0')
	}
	if i < 100 {
		for ; sz > 2; sz-- {
			bs = append(bs, '0')
		}
		return strconv.AppendInt(bs, int64(i), 10)
	}

	digits := 0
	if i < 1000 {
		digits = 3
	} else if i < 10000 {
		digits = 4
	} else {
		digits = int(math.Log10(float64(i))) + 1
	}
	for ; sz > digits; sz-- {
		bs = append(bs, '0')
	}

	return strconv.AppendInt(bs, int64(i), 10)
}
