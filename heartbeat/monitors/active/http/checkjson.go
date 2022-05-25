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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/PaesslerAG/gval"
	"github.com/PaesslerAG/jsonpath"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	"github.com/elastic/beats/v7/libbeat/conditions"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type jsonChecker func(interface{}) bool
type compiledJsonCheck struct {
	description string
	check       jsonChecker
	source      string
}

func checkJson(checks []*jsonResponseCheck) (bodyValidator, error) {
	var expressionChecks []compiledJsonCheck
	var conditionChecks []compiledJsonCheck

	for _, check := range checks {
		if check.Expression != "" {
			eval, err := gval.Full(jsonpath.PlaceholderExtension()).NewEvaluable(check.Expression)
			if err != nil {
				return nil, fmt.Errorf("could not compile gval expression '%s': %w", check.Expression, err)
			}

			checkFn := func(d interface{}) bool {
				matches, err := eval.EvalBool(context.Background(), d)
				if err != nil {
					// Conditions cannot match array root JSON responses
					return false
				}

				return matches
			}

			expressionChecks = append(expressionChecks, compiledJsonCheck{
				description: check.Description,
				check:       checkFn,
				source:      check.Expression,
			})
		} else if check.Condition != nil {
			cfgwarn.Deprecate("8.0.0", "JSON conditions are deprecated, use 'expression' instead.")
			cond, err := conditions.NewCondition(check.Condition)
			if err != nil {
				return nil, fmt.Errorf("could not load JSON condition '%s': %w", check.Description, err)
			}

			checkFn := func(d interface{}) bool {
				ms, ok := d.(map[string]interface{})
				if ok {
					return cond.Check(mapstr.M(ms))
				} else {
					return false
				}
			}
			conditionChecks = append(conditionChecks, compiledJsonCheck{
				description: check.Description,
				check:       checkFn,
				source:      fmt.Sprintf("%v", check.Condition),
			})
		}
	}

	return createJsonCheck(expressionChecks, conditionChecks), nil
}

func createJsonCheck(expressionChecks []compiledJsonCheck, conditionChecks []compiledJsonCheck) bodyValidator {
	return func(_ *http.Response, body string) error {
		var validationFailures []string
		if len(expressionChecks) > 0 {
			decoded, err := decodeJson(body, false)
			if err != nil {
				return err
			}
			validationFailures = append(validationFailures, runCompiledJSONChecks(decoded, expressionChecks)...)
		}

		if len(conditionChecks) > 0 {
			decoded, err := decodeJson(body, true)
			if err != nil {
				// This should only err if the JSON is unparsable,
				// so no need to handle returning 'errs' if this happens
				return err
			}
			validationFailures = append(validationFailures, runCompiledJSONChecks(decoded, conditionChecks)...)
		}

		if len(validationFailures) > 0 {
			bodyTrunc := len(body)
			if bodyTrunc > 2048 {
				bodyTrunc = 2048
			}
			return fmt.Errorf(
				"JSON body did not match %d expressions or conditions '%s'. Received JSON (first 2048 chars): %s",
				len(validationFailures),
				strings.Join(validationFailures, ","),
				body[:bodyTrunc-1], // Only print the first 2k of JSON to limit size
			)
		}

		return nil
	}
}

func decodeJson(body string, forCondition bool) (result interface{}, err error) {
	decoder := json.NewDecoder(strings.NewReader(body))
	// Condition checks need to convert the parsed numeric
	// values in a way appropriate for the condition evaluator. GVal only works if
	// this is not enabled, so expression checks have a separate codepath.
	if forCondition {
		decoder.UseNumber()
	}

	err = decoder.Decode(&result)
	if err != nil {
		return result, fmt.Errorf("could not parse JSON: %w", err)
	}

	if forCondition {
		if resMap, ok := result.(map[string]interface{}); ok {
			jsontransform.TransformNumbers(resMap)
			return resMap, nil
		} else {
			return nil, fmt.Errorf("received non-object JSON for condition, use expression syntax for arrays of JSON instead")
		}
	}

	return result, nil
}

func runCompiledJSONChecks(decodedBody interface{}, compiledChecks []compiledJsonCheck) []string {
	var errorDescs []string
	for _, compiledCheck := range compiledChecks {
		ok := compiledCheck.check(decodedBody)
		if !ok {
			errorDescs = append(errorDescs, fmt.Sprintf("rule '%s'(%s) not matched.", compiledCheck.description, compiledCheck.source))
		}
	}

	return errorDescs
}
