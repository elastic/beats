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

package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	pkgerrors "github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/match"
	"github.com/elastic/beats/libbeat/conditions"
)

type RespCheck func(*http.Response) error

var (
	errBodyMismatch = errors.New("body mismatch")
)

func makeValidateResponse(config *responseParameters) (RespCheck, error) {
	var checks []RespCheck

	if config.Status > 0 {
		checks = append(checks, checkStatus(config.Status))
	} else {
		checks = append(checks, checkStatusOK)
	}

	if len(config.RecvHeaders) > 0 {
		checks = append(checks, checkHeaders(config.RecvHeaders))
	}

	if len(config.RecvBody) > 0 {
		checks = append(checks, checkBody(config.RecvBody))
	}

	if len(config.RecvJSON) > 0 {
		jsonChecks, err := checkJSON(config.RecvJSON)
		if err != nil {
			return nil, err
		}
		checks = append(checks, jsonChecks)
	}

	return checkAll(checks...), nil
}

func checkOK(_ *http.Response) error { return nil }

// TODO: collect all errors into on error message.
func checkAll(checks ...RespCheck) RespCheck {
	switch len(checks) {
	case 0:
		return checkOK
	case 1:
		return checks[0]
	}

	return func(r *http.Response) error {
		for _, check := range checks {
			if err := check(r); err != nil {
				return err
			}
		}
		return nil
	}
}

func checkStatus(status uint16) RespCheck {
	return func(r *http.Response) error {
		if r.StatusCode == int(status) {
			return nil
		}
		return fmt.Errorf("received status code %v expecting %v", r.StatusCode, status)
	}
}

func checkStatusOK(r *http.Response) error {
	if r.StatusCode >= 400 {
		return errors.New(r.Status)
	}
	return nil
}

func checkHeaders(headers map[string]string) RespCheck {
	return func(r *http.Response) error {
		for k, v := range headers {
			value := r.Header.Get(k)
			if v != value {
				return fmt.Errorf("header %v is '%v' expecting '%v' ", k, value, v)
			}
		}
		return nil
	}
}

func checkBody(body []match.Matcher) RespCheck {
	return func(r *http.Response) error {
		content, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return err
		}
		for _, m := range body {
			if m.Match(content) {
				return nil
			}
		}
		return errBodyMismatch
	}
}

func checkJSON(checks []*jsonResponseCheck) (RespCheck, error) {
	type compiledCheck struct {
		description string
		condition   conditions.Condition
	}

	var compiledChecks []compiledCheck

	for _, check := range checks {
		cond, err := conditions.NewCondition(check.Condition)
		if err != nil {
			return nil, err
		}
		compiledChecks = append(compiledChecks, compiledCheck{check.Description, cond})
	}

	return func(r *http.Response) error {
		decoded := &common.MapStr{}
		err := json.NewDecoder(r.Body).Decode(decoded)

		if err != nil {
			body, _ := ioutil.ReadAll(r.Body)
			return pkgerrors.Wrapf(err, "could not parse JSON for body check with condition. Source: %s", body)
		}

		var errorDescs []string
		for _, compiledCheck := range compiledChecks {
			ok := compiledCheck.condition.Check(decoded)
			if !ok {
				errorDescs = append(errorDescs, compiledCheck.description)
			}
		}

		if len(errorDescs) > 0 {
			return fmt.Errorf(
				"JSON body did not match %d conditions '%s' for monitor. Received JSON %+v",
				len(errorDescs),
				strings.Join(errorDescs, ","),
				decoded,
			)
		}

		return nil
	}, nil
}
