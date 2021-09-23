package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/PaesslerAG/gval"
	"github.com/PaesslerAG/jsonpath"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	"github.com/elastic/beats/v7/libbeat/conditions"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type jsonChecker func(interface{}) bool
type compiledJSONCheck struct {
	description string
	check       jsonChecker
	source      string
}

func checkJSON(checks []*jsonResponseCheck) (bodyValidator, error) {
	var conditionChecks []compiledJSONCheck
	var expressionChecks []compiledJSONCheck

	for _, check := range checks {
		if check.Expression != "" && check.Condition != nil {
			return nil, fmt.Errorf("only one of 'expression' and 'condition' can be specified for JSON check '%s'", check.Description)
		}

		if check.Expression != "" {
			eval, err := gval.Full(jsonpath.PlaceholderExtension()).NewEvaluable(check.Expression)
			if err != nil {
				return nil, fmt.Errorf("could not compile gval expression '%s': %w", check.Expression, err)
			}

			checkFn := func(d interface{}) bool {
				matches, err := eval.EvalBool(context.Background(), d)
				if err != nil {
					logp.Warn("error matching JSON against boolean expression: '%v', %v: %v", check.Expression, matches, err)
					return false
				}

				return matches
			}

			logp.Warn("Add expression: %s\n", check.Expression)
			expressionChecks = append(expressionChecks, compiledJSONCheck{
				description: check.Description,
				check:       checkFn,
				source:      check.Expression,
			})
		} else if check.Condition != nil {
			cond, err := conditions.NewCondition(check.Condition)
			if err != nil {
				return nil, fmt.Errorf("could not load JSON condition '%s': %w", check.Description, err)
			}

			checkFn := func(d interface{}) bool {
				ms, ok := d.(map[string]interface{})
				if ok {
					return cond.Check(common.MapStr(ms))
				} else {
					return false
				}
			}
			conditionChecks = append(conditionChecks, compiledJSONCheck{
				description: check.Description,
				check:       checkFn,
				source:      fmt.Sprintf("%v", check.Condition),
			})
		}
	}

	return func(_ *http.Response, body string) error {
		var validationFailures []string
		if len(expressionChecks) > 0 {
			decoded, err := decodeJSON(body, false)
			if err != nil {
				return err
			}
			validationFailures = append(validationFailures, runCompiledJSONChecks(decoded, expressionChecks)...)
		}

		if len(conditionChecks) > 0 {
			decoded, err := decodeJSON(body, true)
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
	}, nil
}

func decodeJSON(body string, forCondition bool) (result interface{}, err error) {
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
			if forCondition {
				jsontransform.TransformNumbers(resMap)
			}
			return interface{}(resMap), nil
		} else {
			return nil, fmt.Errorf("received non-object JSON for condition, use expression syntax for arrays of JSON instead")
		}
	}

	return result, nil
}

func runCompiledJSONChecks(decodedBody interface{}, compiledChecks []compiledJSONCheck) []string {
	var errorDescs []string
	for _, compiledCheck := range compiledChecks {
		ok := compiledCheck.check(decodedBody)
		if !ok {
			errorDescs = append(errorDescs, fmt.Sprintf("rule '%s'(%s) not matched.", compiledCheck.description, compiledCheck.source))
		}
	}

	return errorDescs
}
