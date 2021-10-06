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
	"errors"
	"fmt"
	"net/http"

	"github.com/elastic/beats/v7/heartbeat/reason"
	"github.com/elastic/beats/v7/libbeat/common/match"
)

// multiValidator combines multiple validations of each type into a single easy to use object.
type multiValidator struct {
	respValidators []respValidator
	bodyValidators []bodyValidator
}

func (rv multiValidator) wantsBody() bool {
	return len(rv.bodyValidators) > 0
}

func (rv multiValidator) validate(resp *http.Response, body string) reason.Reason {
	for _, respValidator := range rv.respValidators {
		if err := respValidator(resp); err != nil {
			return reason.ValidateFailed(err)
		}
	}

	for _, bodyValidator := range rv.bodyValidators {
		if err := bodyValidator(resp, body); err != nil {
			return reason.ValidateFailed(err)
		}
	}

	return nil
}

// respValidator is used for validating using only the non-body fields of the *http.Response.
// Accessing the body of the response in such a validator should not be done due, use bodyValidator
// for those purposes instead.
type respValidator func(*http.Response) error

// bodyValidator lets you validate a stringified version of the body along with other metadata in
// *http.Response.
type bodyValidator func(*http.Response, string) error

var (
	errBodyPositiveMismatch  = errors.New("only positive pattern mismatch")
	errBodyNegativeMismatch  = errors.New("only negative pattern mismatch")
	errBodyNoValidCheckType  = errors.New("no valid check type under check.body, only 'positive' or 'negative' is expected")
	errBodyNoValidCheckParam = errors.New("no valid check parameters under check.body")
	errBodyIllegalBody       = errors.New("unsupported content under check.body")
)

func makeValidateResponse(config *responseParameters) (multiValidator, error) {
	var respValidators []respValidator
	var bodyValidators []bodyValidator

	if len(config.Status) > 0 {
		respValidators = append(respValidators, checkStatus(config.Status))
	} else {
		respValidators = append(respValidators, checkStatusOK)
	}

	if len(config.RecvHeaders) > 0 {
		respValidators = append(respValidators, checkHeaders(config.RecvHeaders))
	}

	if config.RecvBody != nil {
		pm, nm, err := parseBody(config.RecvBody)
		if err != nil {
			bodyValidators = append(bodyValidators, func(response *http.Response, body string) error {
				return err
			})
		}
		bodyValidators = append(bodyValidators, checkBody(pm, nm))
	}

	if len(config.RecvJSON) > 0 {
		jsonChecks, err := checkJson(config.RecvJSON)
		if err != nil {
			return multiValidator{}, fmt.Errorf("could not load JSON check: %w", err)
		}
		bodyValidators = append(bodyValidators, jsonChecks)
	}

	return multiValidator{respValidators, bodyValidators}, nil
}

func checkStatus(status []uint16) respValidator {
	return func(r *http.Response) error {
		for _, v := range status {
			if r.StatusCode == int(v) {
				return nil
			}
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

func parseBody(b interface{}) (positiveMatch, negativeMatch []match.Matcher, err error) {
	// run through this code block if there is only string
	if pat, ok := b.(string); ok {
		return append(positiveMatch, match.MustCompile(pat)), negativeMatch, nil
	}

	// run through this code block if there is no positive or negative keyword in response body
	// in this case, there's only plain body
	if p, ok := b.([]interface{}); ok {
		for _, pp := range p {
			if pat, ok := pp.(string); ok {
				positiveMatch = append(positiveMatch, match.MustCompile(pat))
			}
		}
		return positiveMatch, negativeMatch, nil
	}

	// run through this part if there exists positive/negative keyword in response body
	// in this case, there will be 3 possibilities: positive + negative / positive / negative
	if m, ok := b.(map[string]interface{}); ok {
		for checkType, v := range m {
			if checkType != "positive" && checkType != "negative" {
				return positiveMatch, negativeMatch, errBodyNoValidCheckType
			}
			if params, ok := v.([]interface{}); ok {
				for _, param := range params {
					if pat, ok := param.(string); ok {
						if checkType == "positive" {
							positiveMatch = append(positiveMatch, match.MustCompile(pat))
						} else if checkType == "negative" {
							negativeMatch = append(negativeMatch, match.MustCompile(pat))
						}
					}
				}
			}
		}
		return positiveMatch, negativeMatch, nil
	}
	return positiveMatch, negativeMatch, errBodyIllegalBody
}

/* checkBody accepts 2 check types:
1. positive
2. negative
So, there are 4 kinds of scenarios:
1. none of check types
2. only positive
3. only negative
4. positive and negative both here
*/
func checkBody(positiveMatch, negativeMatch []match.Matcher) bodyValidator {
	// in case there's both valid positive and negative regex pattern
	return func(r *http.Response, body string) error {
		if len(positiveMatch) == 0 && len(negativeMatch) == 0 {
			return errBodyNoValidCheckParam
		}
		// positive match loop
		for _, pattern := range positiveMatch {
			// return immediately if there is no negative match
			if pattern.MatchString(body) && len(negativeMatch) == 0 {
				return nil
			}
		}
		// return immediately if there is no negative match
		if len(negativeMatch) == 0 {
			return errBodyPositiveMismatch
		}

		// negative match loop
		for _, pattern := range negativeMatch {
			if pattern.MatchString(body) {
				return errBodyNegativeMismatch
			}
		}
		return nil
	}
}
