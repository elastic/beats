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

package codec

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/dtfmt"
	"github.com/elastic/go-structform"
)

func MakeTimestampEncoder() func(*time.Time, structform.ExtVisitor) error {
	formatter, err := dtfmt.NewFormatter("yyyy-MM-dd'T'HH:mm:ss.SSS'Z'")
	if err != nil {
		panic(err)
	}

	buf := make([]byte, 0, formatter.EstimateSize())
	return func(t *time.Time, v structform.ExtVisitor) error {
		tmp, err := formatter.AppendTo(buf, (*t).UTC())
		if err != nil {
			return err
		}

		buf = tmp[:0]
		return v.OnStringRef(tmp)
	}
}

func MakeBCTimestampEncoder() func(*common.Time, structform.ExtVisitor) error {
	enc := MakeTimestampEncoder()
	return func(t *common.Time, v structform.ExtVisitor) error {
		return enc((*time.Time)(t), v)
	}
}
