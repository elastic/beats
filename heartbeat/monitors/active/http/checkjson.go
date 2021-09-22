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

type jsonChecker func(map[string]interface{}) bool
type compiledJSONCheck struct {
	description string
	check       jsonChecker
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

			checkFn := func(m map[string]interface{}) bool {
				matches, err := eval.EvalBool(context.TODO(), m)
				if err != nil {
					logp.Warn("error matching JSON against boolean expression: '%v', %v: %v", check.Expression, matches, err)
					return false
				}

				return matches
			}

			expressionChecks = append(expressionChecks, compiledJSONCheck{check.Description, checkFn})
		} else if check.Condition != nil {
			cond, err := conditions.NewCondition(check.Condition)
			if err != nil {
				return nil, fmt.Errorf("could not load JSON condition '%s': %w", check.Description, err)
			}

			checkFn := func(ms map[string]interface{}) bool { return cond.Check(common.MapStr(ms)) }
			conditionChecks = append(conditionChecks, compiledJSONCheck{check.Description, checkFn})
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

func decodeJSON(body string, transformNumber bool) (map[string]interface{}, error) {
	decoded := &map[string]interface{}{}
	decoder := json.NewDecoder(strings.NewReader(body))

	// Condition checks require useNumber to be true to convert the parsed numeric
	// values in a way appropriate for the condition evaluator. GVal only works if
	// this is not enabled, so expression checks have a separate codepath.
	if transformNumber {
		decoder.UseNumber()
	}
	err := decoder.Decode(decoded)

	if transformNumber {
		jsontransform.TransformNumbers(*decoded)
	}

	if err != nil {
		return nil, fmt.Errorf("could not parse JSON for body check with condition. Err: %w Source: %s", err, body)
	}

	return *decoded, nil
}

func runCompiledJSONChecks(decodedBody map[string]interface{}, compiledChecks []compiledJSONCheck) []string {
	var errorDescs []string
	for _, compiledCheck := range compiledChecks {
		ok := compiledCheck.check(decodedBody)
		if !ok {
			errorDescs = append(errorDescs, fmt.Sprintf("rule '%s' not matched", compiledCheck.description))
		}
	}

	return errorDescs
}
