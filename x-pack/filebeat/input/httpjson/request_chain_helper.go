// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	retryablehttp "github.com/hashicorp/go-retryablehttp"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

const (
	// This is generally updated with chain responses, if present, as they continue to occur
	// Otherwise this is always the last response of the root request w.r.t pagination
	lastResponse = "last_response"
	// This is always the first root response
	firstResponse = "first_response"
	// This is always the last response of the parent (root) request w.r.t pagination
	// This is only set if chaining is used
	parentLastResponse = "parent_last_response"
)

func newChainHTTPClient(ctx context.Context, authCfg *authConfig, requestCfg *requestConfig, log *logp.Logger, reg *monitoring.Registry, p ...*Policy) (*httpClient, error) {
	// Make retryable HTTP client
	netHTTPClient, err := newNetHTTPClient(ctx, requestCfg, log, reg)
	if err != nil {
		return nil, err
	}

	var retryPolicyFunc retryablehttp.CheckRetry
	if len(p) != 0 {
		retryPolicyFunc = p[0].CustomRetryPolicy
	} else {
		retryPolicyFunc = retryablehttp.DefaultRetryPolicy
	}

	client := &retryablehttp.Client{
		HTTPClient:   netHTTPClient,
		Logger:       newRetryLogger(log),
		RetryWaitMin: requestCfg.Retry.getWaitMin(),
		RetryWaitMax: requestCfg.Retry.getWaitMax(),
		RetryMax:     requestCfg.Retry.getMaxAttempts(),
		CheckRetry:   retryPolicyFunc,
		Backoff:      retryablehttp.DefaultBackoff,
	}

	limiter := newRateLimiterFromConfig(requestCfg.RateLimit, log)

	if authCfg != nil && authCfg.OAuth2.isEnabled() {
		authClient, err := authCfg.OAuth2.client(ctx, client.StandardClient())
		if err != nil {
			return nil, err
		}
		return &httpClient{client: authClient, limiter: limiter}, nil
	}

	return &httpClient{client: client.StandardClient(), limiter: limiter}, nil
}

func evaluateResponse(expression *valueTpl, data []byte, log *logp.Logger) (bool, error) {
	var dataMap mapstr.M

	err := json.Unmarshal(data, &dataMap)
	if err != nil {
		return false, fmt.Errorf("error while unmarshalling data : %w", err)
	}
	tr := transformable{}
	paramCtx := &transformContext{
		firstEvent:    &mapstr.M{},
		lastEvent:     &mapstr.M{},
		firstResponse: &response{},
		lastResponse:  &response{body: dataMap},
	}

	val, err := expression.Execute(paramCtx, tr, "", nil, log)
	if err != nil {
		return false, fmt.Errorf("error while evaluating expression : %w", err)
	}
	result, err := strconv.ParseBool(val)
	if err != nil {
		return false, fmt.Errorf("error while parsing boolean value of string : %w", err)
	}

	return result, nil
}

// fetchValueFromContext evaluates a given expression and returns the appropriate value from context variables if present
func fetchValueFromContext(trCtx *transformContext, expression string) (string, bool, error) {
	var val interface{}

	switch keys := processExpression(expression); keys[0] {
	case lastResponse:
		respMap, err := responseToMap(trCtx.lastResponse)
		if err != nil {
			return "", false, err
		}
		val, err = iterateRecursive(respMap, keys[1:], 0)
		if err != nil {
			return "", false, err
		}
	case parentLastResponse:
		respMap, err := responseToMap(trCtx.parentTrCtx.lastResponse)
		if err != nil {
			return "", false, err
		}
		val, err = iterateRecursive(respMap, keys[1:], 0)
		if err != nil {
			return "", false, err
		}
	case firstResponse:
		// since first response body is already a map, we do not need to transform it
		respMap, err := responseToMap(trCtx.firstResponse)
		if err != nil {
			return "", false, err
		}
		val, err = iterateRecursive(respMap, keys[1:], 0)
		if err != nil {
			return "", false, err
		}
	// In this scenario we treat the expression as a hardcoded value, with which we will replace the fixed-pattern
	case expression:
		return expression, true, nil
	default:
		return "", false, fmt.Errorf("context value not supported for key: %q in expression %q", keys[0], expression)
	}

	return fmt.Sprint(val), true, nil
}

func responseToMap(r *response) (mapstr.M, error) {
	if r.body == nil {
		return nil, fmt.Errorf("response body is empty for request url: %s", &r.url)
	}
	respMap := map[string]interface{}{
		"header": make(mapstr.M),
		"body":   make(mapstr.M),
	}

	for key, value := range r.header {
		respMap["header"] = mapstr.M{
			key: value,
		}
	}
	respMap["body"] = r.body

	return respMap, nil
}

func iterateRecursive(m mapstr.M, keys []string, depth int) (interface{}, error) {
	val := m[keys[depth]]

	if val == nil {
		return nil, fmt.Errorf("value of expression could not be determined for key %s", strings.Join(keys[:depth+1], "."))
	}

	switch v := reflect.ValueOf(val); v.Kind() {
	case reflect.Bool:
		return v.Bool(), nil
	case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
		return v.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint64:
		return v.Uint(), nil
	case reflect.Float32, reflect.Float64:
		return v.Float(), nil
	case reflect.String:
		return v.String(), nil
	case reflect.Map:
		nextMap, ok := v.Interface().(map[string]interface{})
		if !ok {
			return nil, errors.New("unable to parse the value of the given expression")
		}
		depth++
		if depth >= len(keys) {
			return nil, errors.New("value of expression could not be determined")
		}
		return iterateRecursive(nextMap, keys, depth)
	default:
		return nil, fmt.Errorf("unable to parse the value of the expression %s: type %T is not handled", strings.Join(keys[:depth+1], "."), val)
	}
}

// processExpression, splits the expression string based on the separator and looks for
// supported keywords. If present, returns an expression array containing separated elements.
// If no keywords are present, the expression is treated as a hardcoded value and returned
// as a merged string which is the only array element.
func processExpression(expression string) []string {
	if !strings.HasPrefix(expression, ".") {
		return []string{expression}
	}
	switch {
	case strings.HasPrefix(expression, "."+firstResponse+"."),
		strings.HasPrefix(expression, "."+lastResponse+"."),
		strings.HasPrefix(expression, "."+parentLastResponse+"."):
		return strings.Split(expression, ".")[1:]
	default:
		return []string{expression}
	}
}

func tryAssignAuth(parentConfig *authConfig, childConfig *authConfig) *authConfig {
	if parentConfig != nil && childConfig == nil {
		return parentConfig
	}
	return childConfig
}
