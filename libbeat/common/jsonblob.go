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

package common

import (
	"encoding/json"
	"errors"
)

// JSONBlob is a custom type that can unpack raw JSON strings or objects into
// a json.RawMessage.
type JSONBlob json.RawMessage

func (b *JSONBlob) Unpack(v interface{}) error {
	switch t := v.(type) {
	case string:
		*b = []byte(t)
	default:
		m, err := json.Marshal(v)
		if err != nil {
			return err
		}
		*b = m
	}

	if !json.Valid(*b) {
		return errors.New("the field can't be converted to valid JSON")
	}

	return nil
}
