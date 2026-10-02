// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package billing

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"

	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
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

// retryAfterHeaderName is the standard header we write the longest retry hint to.
const retryAfterHeaderName = "Retry-After"

// standardRetryAfterHeaders maps the (lower cased) headers azcore's retry policy
// reads to a parser for their value. azcore uses the first one present, checking
// Retry-After-Ms, then x-ms-retry-after-ms, then Retry-After, regardless of which
// one asks for the longest wait.
var standardRetryAfterHeaders = map[string]func(string) (time.Duration, bool){
	"retry-after":         parseRetryAfter,
	"retry-after-ms":      parseRetryAfterMilliseconds,
	"x-ms-retry-after-ms": parseRetryAfterMilliseconds,
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
// When the service does send hints we wait for the longest one, up to MaxRetryDelay.
//
// There is plenty of headroom: the metricset period is 1440m, mb defaults Timeout to
// Period, and the service issues its calls with context.Background().
const (
	billingRetryDelay    = 10 * time.Second
	billingMaxRetryDelay = 5 * time.Minute
	billingMaxRetries    = int32(5)

	// maxParsableRetryAfter bounds header values before they are turned into a
	// time.Duration. Anything larger is nonsensical and would risk overflowing.
	maxParsableRetryAfter = 24 * time.Hour
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

// retryAfterTranslationPolicy folds every retry hint on an error response, standard
// or Azure Cost Management / Consumption specific, into a single Retry-After header
// carrying the longest wait, so azcore's retry policy honours it instead of falling
// back to its own exponential backoff or retrying before the quota has cleared.
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

// translateRetryAfter replaces every retry hint on an error response with a single
// Retry-After carrying the longest wait.
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

	// A throttled response can carry several hints, e.g. a standard Retry-After next
	// to a Cost Management one, each describing a different limit. A retry can only
	// succeed once all of them have cleared, so wait for the longest. Left alone,
	// azcore would use the first standard header it finds, however short, and could
	// spend the whole retry budget before the longer limit clears.
	name, delay := longestRetryAfter(resp.Header)
	if delay <= 0 {
		return
	}

	// azcore abandons the request outright when Retry-After exceeds MaxRetryDelay.
	// Retrying after the cap is strictly better than dropping the day's data: if the
	// quota has not cleared yet we simply get another 429 with a fresh hint.
	if delay > billingMaxRetryDelay {
		delay = billingMaxRetryDelay
	}

	// azcore reads the millisecond headers before Retry-After, so they have to go
	// for the value set below to be the one it uses.
	deleteStandardRetryAfter(resp.Header)

	seconds := int(math.Ceil(delay.Seconds()))
	resp.Header.Set(retryAfterHeaderName, strconv.Itoa(seconds))

	if p.log != nil {
		p.log.Debugw(
			"using the longest Azure retry hint as the Retry-After header",
			"http.response.status_code", resp.StatusCode,
			"azure.billing.rate_limit.header", name,
			"azure.billing.rate_limit.retry_after_seconds", seconds,
		)
	}
}

// longestRetryAfter returns the longest wait advertised by any standard or Azure rate
// limit "retry after" header, together with the name of the header it came from. It
// returns a zero duration when no usable header is present.
func longestRetryAfter(header http.Header) (string, time.Duration) {
	var (
		maxName  string
		maxDelay time.Duration
	)

	for name, values := range header {
		parse := retryAfterParser(name)
		if parse == nil {
			continue
		}

		for _, value := range values {
			delay, ok := parse(value)
			if !ok || delay <= maxDelay {
				continue
			}

			maxName, maxDelay = name, delay
		}
	}

	return maxName, maxDelay
}

// retryAfterParser returns the parser for the value of a header carrying a retry
// hint, or nil when the header carries none. Names are compared case insensitively
// because http.Header canonicalises keys.
func retryAfterParser(name string) func(string) (time.Duration, bool) {
	lower := strings.ToLower(name)
	if parse, ok := standardRetryAfterHeaders[lower]; ok {
		return parse
	}

	if strings.HasPrefix(lower, rateLimitRetryAfterHeaderPrefix) &&
		strings.HasSuffix(lower, rateLimitRetryAfterHeaderSuffix) {
		return parseRetryAfterSeconds
	}

	return nil
}

// deleteStandardRetryAfter removes every header azcore's retry policy reads, whatever
// the casing of its key.
func deleteStandardRetryAfter(header http.Header) {
	for name := range header {
		if _, ok := standardRetryAfterHeaders[strings.ToLower(name)]; ok {
			delete(header, name)
		}
	}
}

// parseRetryAfter parses a standard Retry-After value, which is either a number of
// seconds or an HTTP date.
func parseRetryAfter(value string) (time.Duration, bool) {
	if delay, ok := parseRetryAfterSeconds(value); ok {
		return delay, true
	}

	date, err := http.ParseTime(strings.TrimSpace(value))
	if err != nil {
		return 0, false
	}

	delay := time.Until(date)
	if delay <= 0 {
		return 0, false
	}

	return min(delay, maxParsableRetryAfter), true
}

// parseRetryAfterSeconds parses a header value expressed in seconds.
func parseRetryAfterSeconds(value string) (time.Duration, bool) {
	return parseRetryAfterNumber(value, time.Second)
}

// parseRetryAfterMilliseconds parses a header value expressed in milliseconds.
func parseRetryAfterMilliseconds(value string) (time.Duration, bool) {
	return parseRetryAfterNumber(value, time.Millisecond)
}

// parseRetryAfterNumber parses a header value expressed as a number of units.
// Surrounding whitespace is tolerated; anything that is not a positive number is
// ignored so a malformed header can never break or stall a fetch.
func parseRetryAfterNumber(value string, unit time.Duration) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	// Be lenient: the headers are documented as integers, but accept decimals too.
	n, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(n) || n <= 0 {
		return 0, false
	}

	// Bound the value before converting so an absurd header cannot overflow the
	// duration into something negative. The caller clamps to MaxRetryDelay anyway.
	if n > float64(maxParsableRetryAfter/unit) {
		return maxParsableRetryAfter, true
	}

	return time.Duration(n * float64(unit)), true
}
