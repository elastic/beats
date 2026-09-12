// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package billing

import (
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Azure Cost Management and Consumption throttle aggressively (per tenant, per
// entity, per client type, and on a query-processing-unit budget) and answer with
// HTTP 429. The wait hint comes back in vendor specific headers:
//
//	x-ms-ratelimit-microsoft.costmanagement-qpu-retry-after
//	x-ms-ratelimit-microsoft.costmanagement-entity-retry-after
//	x-ms-ratelimit-microsoft.costmanagement-tenant-retry-after
//	x-ms-ratelimit-microsoft.costmanagement-client-retry-after
//	x-ms-ratelimit-microsoft.consumption-retry-after
//	x-ms-ratelimit-microsoft.consumption-clientappid-retry-after  (observed live)
//	x-ms-ratelimit-microsoft.consumption-tenant-retry-after       (observed live)
//
// azcore's retry policy only understands "Retry-After", "Retry-After-Ms" and
// "x-ms-retry-after-ms" (see shared.RetryAfter), so without translation it ignores
// the service hint and falls back to its own short exponential backoff.
//
// Instead of enumerating the header names (Azure adds new limit dimensions without
// notice) we match any header with the vendor rate limit prefix and the retry-after
// suffix below. That covers all the names above plus future variants.
const (
	rateLimitRetryAfterHeaderPrefix = "x-ms-ratelimit-microsoft."
	rateLimitRetryAfterHeaderSuffix = "-retry-after"
)

// retryAfterHeaderName is the standard header azcore's retry policy reads and the
// one we synthesise from Azure's vendor specific hints.
const retryAfterHeaderName = "Retry-After"

// standardRetryAfterHeaders lists (lower cased) the headers azcore's retry policy
// already reads. If the service sent one of them we must not touch anything: azcore
// is already going to honour it, and overwriting it could shorten the requested wait.
var standardRetryAfterHeaders = []string{
	"retry-after",
	"retry-after-ms",
	"x-ms-retry-after-ms",
}

// Cost Management applies part of its throttling per "client type". Callers that do
// not identify themselves share a single default quota pool with every other
// unidentified caller in the tenant, so a noisy neighbour can exhaust our budget.
// Sending a ClientType header moves us into our own pool.
//
// The header is not part of the published REST reference; it is the mitigation
// documented by Microsoft in the Cost Management throttling guidance / support
// answers for HTTP 429. Header names are case insensitive on the wire, but we set
// the map key directly (instead of Header.Set, which would canonicalise it to
// "Clienttype") so the exact documented spelling reaches the service.
const (
	clientTypeHeaderName  = "ClientType"
	clientTypeHeaderValue = "elastic-metricbeat-azure-billing"
)

// Retry budget.
//
// azcore's defaults (3 retries, 800ms base, 60s cap) total roughly 7-11s, which is
// useless against a per-minute quota, and it gives up entirely when a Retry-After
// exceeds MaxRetryDelay. We therefore raise all three.
//
// The fallback delay used when the service sends no hint is
// ((2^try)-1) * RetryDelay * jitter, jitter in [0.8, 1.3), capped at MaxRetryDelay:
//
//	try 1: 10s  (max 13s)
//	try 2: 30s  (max 39s)
//	try 3: 70s  (max 91s)
//	try 4: 150s (max 195s)
//	try 5: 310s (capped at 300s)
//
// Worst case total wait is therefore ~638s (~10m40s) spread over 6 attempts, which
// is enough for the 1 minute and most of the 10 second quota windows to refill.
// When the service does send a hint we honour it verbatim up to MaxRetryDelay.
//
// There is plenty of headroom: the metricset period is 1440m, mb defaults Timeout to
// Period, and the service issues its calls with context.Background().
const (
	billingRetryDelay    = 10 * time.Second
	billingMaxRetryDelay = 5 * time.Minute
	billingMaxRetries    = int32(5)

	// maxParsableRetryAfterSeconds bounds header values before they are turned into
	// a time.Duration. Anything larger is nonsensical and would risk overflowing.
	maxParsableRetryAfterSeconds = float64(24 * 60 * 60)
)

// newRetryOptions returns the retry configuration shared by both billing clients.
//
// StatusCodes is deliberately left unset so azcore keeps its defaults
// (408, 429, 500, 502, 503, 504); 429 is the one we care about here.
func newRetryOptions() policy.RetryOptions {
	return policy.RetryOptions{
		MaxRetries:    billingMaxRetries,
		RetryDelay:    billingRetryDelay,
		MaxRetryDelay: billingMaxRetryDelay,
	}
}

// newClientOptions builds the ARM client options shared by the usage details and
// forecast clients.
func newClientOptions(config azure.Config, logger *logp.Logger) arm.ClientOptions {
	return arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Cloud: azure.BuildCloudConfig(config),
			Retry: newRetryOptions(),
			// Runs once per request, before the retry policy.
			PerCallPolicies: []policy.Policy{clientTypePolicy{}},
			// Runs inside the retry loop, after the retry policy, so it sees every
			// try's response before the retry policy inspects it for a wait hint.
			PerRetryPolicies: []policy.Policy{newRetryAfterTranslationPolicy(logger)},
		},
	}
}

// clientTypePolicy stamps the Cost Management ClientType header on every request.
type clientTypePolicy struct{}

func (clientTypePolicy) Do(req *policy.Request) (*http.Response, error) {
	// Assign the map key directly to preserve the documented header casing.
	req.Raw().Header[clientTypeHeaderName] = []string{clientTypeHeaderValue}

	return req.Next()
}

// retryAfterTranslationPolicy copies the Azure Cost Management / Consumption rate
// limit wait hint into the standard Retry-After header so azcore's retry policy
// honours it instead of falling back to its own exponential backoff.
type retryAfterTranslationPolicy struct {
	log *logp.Logger
}

func newRetryAfterTranslationPolicy(logger *logp.Logger) *retryAfterTranslationPolicy {
	p := &retryAfterTranslationPolicy{}
	if logger != nil {
		p.log = logger.Named("azure billing retry")
	}

	return p
}

func (p *retryAfterTranslationPolicy) Do(req *policy.Request) (*http.Response, error) {
	resp, err := req.Next()
	if err != nil {
		return resp, err
	}

	p.translateRetryAfter(resp)

	return resp, nil
}

// translateRetryAfter sets Retry-After from the vendor rate limit headers, if needed.
func (p *retryAfterTranslationPolicy) translateRetryAfter(resp *http.Response) {
	if resp == nil || resp.Header == nil {
		return
	}

	// Only error responses can be retried. Restricting the rewrite to them keeps us
	// from advertising a Retry-After on successful responses (Cost Management also
	// reports its remaining budget on 200s). We deliberately do not restrict this to
	// 429 so throttling surfaced as 503 or 500 is handled too.
	if resp.StatusCode < http.StatusBadRequest {
		return
	}

	// Never overwrite a hint azcore can already read.
	if hasStandardRetryAfter(resp.Header) {
		return
	}

	name, delay := maxRateLimitRetryAfter(resp.Header)
	if delay <= 0 {
		return
	}

	// azcore abandons the request outright when Retry-After exceeds MaxRetryDelay.
	// Retrying after the cap is strictly better than dropping the day's data: if the
	// quota has not cleared yet we simply get another 429 with a fresh hint.
	if delay > billingMaxRetryDelay {
		delay = billingMaxRetryDelay
	}

	seconds := int(math.Ceil(delay.Seconds()))
	resp.Header.Set(retryAfterHeaderName, strconv.Itoa(seconds))

	if p.log != nil {
		p.log.Debugw(
			"translating Azure rate limit retry hint into a Retry-After header",
			"http.response.status_code", resp.StatusCode,
			"azure.billing.rate_limit.header", name,
			"azure.billing.rate_limit.retry_after_seconds", seconds,
		)
	}
}

// hasStandardRetryAfter reports whether the response already carries a non-empty
// "retry after" header that azcore's retry policy understands. Header keys are
// compared case insensitively: http.Header canonicalises keys on the wire, but
// azcore looks some of them up in non canonical form.
func hasStandardRetryAfter(header http.Header) bool {
	for name, values := range header {
		lower := strings.ToLower(name)
		if !slices.Contains(standardRetryAfterHeaders, lower) {
			continue
		}

		for _, value := range values {
			if strings.TrimSpace(value) != "" {
				return true
			}
		}
	}

	return false
}

// maxRateLimitRetryAfter returns the longest wait advertised by any Azure rate limit
// "retry after" header, together with the name of the header it came from. It returns
// a zero duration when no usable header is present.
func maxRateLimitRetryAfter(header http.Header) (string, time.Duration) {
	var (
		maxName  string
		maxDelay time.Duration
	)

	for name, values := range header {
		// http.Header canonicalises keys, so compare case insensitively.
		lower := strings.ToLower(name)
		if !strings.HasPrefix(lower, rateLimitRetryAfterHeaderPrefix) ||
			!strings.HasSuffix(lower, rateLimitRetryAfterHeaderSuffix) {
			continue
		}

		for _, value := range values {
			delay, ok := parseRetryAfterSeconds(value)
			if !ok || delay <= maxDelay {
				continue
			}

			maxName, maxDelay = name, delay
		}
	}

	return maxName, maxDelay
}

// parseRetryAfterSeconds parses a rate limit header value expressed in seconds.
// Surrounding whitespace is tolerated; anything that is not a positive number is
// ignored so a malformed header can never break or stall a fetch.
func parseRetryAfterSeconds(value string) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	// Be lenient: the service documents integer seconds, but accept decimals too.
	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(seconds) || seconds <= 0 {
		return 0, false
	}

	// Bound the value before converting so an absurd header cannot overflow the
	// duration into something negative. The caller clamps to MaxRetryDelay anyway.
	if seconds > maxParsableRetryAfterSeconds {
		seconds = maxParsableRetryAfterSeconds
	}

	return time.Duration(seconds * float64(time.Second)), true
}
