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

package esutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/elastic/go-elasticsearch/v8/esapi"
)

func ToJsonRdr(i interface{}) (io.Reader, error) {
	b, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(b), nil
}

func CheckResp(r *esapi.Response, argErr error) error {
	if argErr != nil {
		return argErr
	}
	if r.StatusCode >= 300 {
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(r.Body)
		if err != nil {
			buf.WriteString(fmt.Sprintf("<error reading body string: %s>", err))
		}
		return fmt.Errorf("bad status code for response(%d): %s", r.StatusCode, buf.String())
	}
	return nil
}

func CheckRetResp(r *esapi.Response, argErr error) (body []byte, err error) {
	if argErr != nil {
		return nil, argErr
	}
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(r.Body)
	if err != nil {
		return nil, fmt.Errorf("<error reading body string: %w>", err)
	}

	if r.StatusCode >= 300 {
		return nil, fmt.Errorf("bad status code for response(%d): %s", r.StatusCode, buf.String())
	}

	return buf.Bytes(), err
}
