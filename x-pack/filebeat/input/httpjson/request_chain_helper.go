// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	retryablehttp "github.com/hashicorp/go-retryablehttp"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
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
		firstEvent:   &mapstr.M{},
		lastEvent:    &mapstr.M{},
		lastResponse: &response{body: dataMap},
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

func tryAssignAuth(parentConfig *authConfig, childConfig *authConfig) *authConfig {
	if parentConfig != nil && childConfig == nil {
		return parentConfig
	}
	return childConfig
}
