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

package memlog

import (
	"io"

	"github.com/menderesk/go-structform/gotype"
	"github.com/menderesk/go-structform/json"
)

type jsonEncoder struct {
	out    io.Writer
	folder *gotype.Iterator
}

func newJSONEncoder(out io.Writer) *jsonEncoder {
	e := &jsonEncoder{out: out}
	e.reset()
	return e
}

func (e *jsonEncoder) reset() {
	visitor := json.NewVisitor(e.out)
	visitor.SetEscapeHTML(false)

	var err error

	// create new encoder with custom time.Time encoding
	e.folder, err = gotype.NewIterator(visitor)
	if err != nil {
		panic(err)
	}
}

func (e *jsonEncoder) Encode(v interface{}) error {
	return e.folder.Fold(v)
}
