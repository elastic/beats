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

// +build gofuzz

package model

import (
	"bytes"
	"encoding/json"

	"go.elastic.co/apm/internal/apmschema"
	"go.elastic.co/fastjson"
)

func Fuzz(data []byte) int {
	type Payload struct {
		Service      *Service      `json:"service"`
		Process      *Process      `json:"process,omitempty"`
		System       *System       `json:"system,omitempty"`
		Errors       []*Error      `json:"errors"`
		Transactions []Transaction `json:"transactions"`
	}

	var payload Payload
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return -1
	}
	raw := make(map[string]interface{})
	if err := json.Unmarshal(data, &raw); err != nil {
		return -1
	}

	if len(payload.Errors) != 0 {
		payload := ErrorsPayload{
			Service: payload.Service,
			Process: payload.Process,
			System:  payload.System,
			Errors:  payload.Errors,
		}
		var w fastjson.Writer
		if err := payload.MarshalFastJSON(&w); err != nil {
			panic(err)
		}
		if err := apmschema.Errors.Validate(bytes.NewReader(w.Bytes())); err != nil {
			panic(err)
		}
	}

	if len(payload.Transactions) != 0 {
		payload := TransactionsPayload{
			Service:      payload.Service,
			Process:      payload.Process,
			System:       payload.System,
			Transactions: payload.Transactions,
		}
		var w fastjson.Writer
		if err := payload.MarshalFastJSON(&w); err != nil {
			panic(err)
		}
		if err := apmschema.Transactions.Validate(bytes.NewReader(w.Bytes())); err != nil {
			panic(err)
		}
	}
	return 0
}
