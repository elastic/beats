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

package mgr

import (
	"encoding/json"
	"fmt"

	"errors"
)

// Request stores either or finished command result.
type Request struct {
	HasFailed bool     `json:"has_failed"`
	Finished  []Result `json:"finished"`
	Failed    []Result `json:"failed"`
}

// Result stores ceph command output (and status).
type Result struct {
	Command string `json:"command"`
	Outb    string `json:"outb"`
	Outs    string `json:"outs"`
}

// UnmarshalResponse method unmarshals the content to the given response object.
func UnmarshalResponse(content []byte, response interface{}) error {
	var request Request
	err := json.Unmarshal(content, &request)
	if err != nil {
		return fmt.Errorf("could not get request data: %w", err)
	}

	if request.HasFailed {
		if len(request.Failed) != 1 {
			return errors.New("expected single failed command")
		}
		return fmt.Errorf("%s: %s", request.Failed[0].Outs, request.Failed[0].Command)
	}

	if len(request.Finished) != 1 {
		return errors.New("expected single finished command")
	}

	err = json.Unmarshal([]byte(request.Finished[0].Outb), response)
	if err != nil {
		return fmt.Errorf("could not get response data: %w", err)
	}
	return nil
}
