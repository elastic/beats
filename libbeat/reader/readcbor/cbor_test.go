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

package readcbor

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ugorji/go/codec"

	"github.com/elastic/beats/libbeat/common"
)

func TestUnmarshal(t *testing.T) {
	t.Run("Log Decode", func(t *testing.T) {
		var h = &codec.CborHandle{}
		var bs []byte

		v := common.MapStr{"level": "info", "name": "testMessage", "time": "2019-01-10T11:11:36-08:00", "message": "This is a test message"}
		codec.NewEncoderBytes(&bs, h).MustEncode(v)

		reader := bytes.NewReader(bs)
		config := &Config{MessageKey: "cbor", KeysUnderRoot: true, AddErrorKey: false}
		r := New(reader, config)
		for {
			msg, err := r.Next()
			if err != nil {
				if err != io.EOF {
					panic(err)
				}
				break
			}
			equal := reflect.DeepEqual(v, msg.Fields["cbor"])
			if !equal {
				fmt.Printf("No Match v value : %v\n type: %T\n, msg value %v\n type: %T\n", v, v, msg, msg)
			}
			assert.Equal(t, equal, true)
		}
	})
}
