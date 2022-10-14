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
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

const (
	lastResponse  = "last_response"
	firstResponse = "first_response"
)

func newChainHTTPClient(ctx context.Context, authCfg *authConfig, requestCfg *requestConfig, log *logp.Logger, p ...*Policy) (*httpClient, error) {
	// Make retryable HTTP client
	netHTTPClient, err := requestCfg.Transport.Client(
		httpcommon.WithAPMHTTPInstrumentation(),
		httpcommon.WithKeepaliveSettings{Disable: true},
	)
	if err != nil {
		return nil, err
	}

	netHTTPClient.CheckRedirect = checkRedirect(requestCfg, log)

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

	val, err := expression.Execute(paramCtx, tr, nil, log)
	if err != nil {
		return false, fmt.Errorf("error while evaluating expression : %w", err)
	}
	result, err := strconv.ParseBool(val)
	if err != nil {
		return false, fmt.Errorf("error while parsing boolean value of string : %w", err)
	}

	return result, nil
}

func fetchValueFromContext(trCtx *transformContext, expression string) (string, bool, error) {
	var val interface{}

	keys := strings.Split(expression, ".")
	if keys[0] == lastResponse {
		respMap, err := responseToMap(trCtx.lastResponse, true)
		if err != nil {
			return "", false, err
		}
		val, err = iterateRecursive(respMap, keys[1:], 0)
		if err != nil {
			return "", false, err
		}
	} else if keys[0] == firstResponse {
		// since first response body is already a map, we do not need to transform it
		respMap, err := responseToMap(trCtx.firstResponse, false)
		if err != nil {
			return "", false, err
		}
		val, err = iterateRecursive(respMap, keys[1:], 0)
		if err != nil {
			return "", false, err
		}
	}
	returnVal := fmt.Sprintf("%v", val)

	return returnVal, true, nil
}

func responseToMap(r *response, mapBody bool) (mapstr.M, error) {
	respMap := map[string]interface{}{
		"header": make(mapstr.M),
		"body":   make(mapstr.M),
	}

	for key, value := range r.header {
		respMap["header"] = mapstr.M{
			key: value,
		}
	}
	if mapBody {
		var bodyMap mapstr.M
		err := json.Unmarshal(r.body.([]byte), &bodyMap)
		if err != nil {
			return nil, err
		}
		respMap["body"] = bodyMap
	} else {
		respMap["body"] = r.body
	}

	return respMap, nil
}

func iterateRecursive(m mapstr.M, keys []string, depth int) (interface{}, error) {
	if m[keys[depth]] == nil {
		return nil, errors.New("value of expression could not be determined")
	}

	val := m[keys[depth]]

	v := reflect.ValueOf(val)
	switch v.Kind() {
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
		nextMap, ok := v.Interface().(mapstr.M)
		if !ok {
			return nil, errors.New("unable to parse the value of the given expression")
		}
		depth = depth + 1
		if len(keys) == depth {
			return nil, errors.New("value of expression could not be determined")
		}
		return iterateRecursive(nextMap, keys, depth)
	default:
		return nil, errors.New("unable to parse the value of the given expression")
	}
}

func tryAssignAuth(parentConfig *authConfig, childConfig *authConfig) *authConfig {
	if parentConfig != nil && childConfig == nil {
		return parentConfig
	}
	return childConfig
}
