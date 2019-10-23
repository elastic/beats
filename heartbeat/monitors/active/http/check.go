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
	"github.com/elastic/beats/libbeat/common/jsontransform"
	"github.com/elastic/beats/libbeat/common/match"
	"github.com/elastic/beats/libbeat/conditions"
)

type comboValidator struct {
	respValidators []respValidator
	bodyValidators []bodyValidator
}

func (rv comboValidator) wantsBody() bool {
	return len(rv.bodyValidators) > 0
}

func (rv comboValidator) validate(resp *http.Response, body string) error {
	for _, respValidator := range rv.respValidators {
		if err := respValidator(resp); err != nil {
			return err
		}
	}

	for _, bodyValidator := range rv.bodyValidators {
		if err := bodyValidator(resp, body); err != nil {
			return err
		}
	}

	return nil
}

type respValidator func(*http.Response) error

type bodyValidator func(*http.Response, string) error

var (
	errBodyMismatch = errors.New("body mismatch")
)

func makeValidateResponse(config *responseParameters) (comboValidator, error) {
	var respValidators []respValidator
	var bodyValidators []bodyValidator

	if config.Status > 0 {
		respValidators = append(respValidators, checkStatus(config.Status))
	} else {
		respValidators = append(respValidators, checkStatusOK)
	}

	if len(config.RecvHeaders) > 0 {
		respValidators = append(respValidators, checkHeaders(config.RecvHeaders))
	}

	if len(config.RecvBody) > 0 {
		bodyValidators = append(bodyValidators, checkBody(config.RecvBody))
	}

	if len(config.RecvJSON) > 0 {
		jsonChecks, err := checkJSON(config.RecvJSON)
		if err != nil {
			return comboValidator{}, err
		}
		bodyValidators = append(bodyValidators, jsonChecks)
	}

	return comboValidator{respValidators, bodyValidators}, nil
}

func checkStatus(status uint16) respValidator {
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

func checkHeaders(headers map[string]string) respValidator {
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

func checkBody(matcher []match.Matcher) bodyValidator {
	return func(r *http.Response, body string) error {
		for _, m := range matcher {
			if m.MatchString(body) {
				return nil
			}
		}
		return errBodyMismatch
	}
}

func checkJSON(checks []*jsonResponseCheck) (bodyValidator, error) {
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

	return func(r *http.Response, body string) error {
		decoded := &common.MapStr{}
		decoder := json.NewDecoder(strings.NewReader(body))
		decoder.UseNumber()
		err := decoder.Decode(decoded)

		if err != nil {
			body, _ := ioutil.ReadAll(r.Body)
			return pkgerrors.Wrapf(err, "could not parse JSON for body check with condition. Source: %s", body)
		}

		jsontransform.TransformNumbers(*decoded)

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
