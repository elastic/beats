/*
Package estransport provides the transport layer for the Elasticsearch client.

It is automatically included in the client provided by the github.com/elastic/go-elasticsearch package
and is not intended for direct use: to configure the client, use the elasticsearch.Config struct.

The default HTTP transport of the client is http.Transport; use the Transport option to customize it;
see the _examples/coniguration.go and _examples/customization.go files in this repository for information.

The package will automatically retry requests on network-related errors, and on specific
response status codes (by default 502, 503, 504). Use the RetryOnStatus option to customize the list.
The transport will not retry a timeout network error, unless enabled by setting EnableRetryOnTimeout to true.

Use the MaxRetries option to configure the number of retries, and set DisableRetry to true
to disable the retry behaviour altogether.

By default, the retry will be performed without any delay; to configure a backoff interval,
implement the RetryBackoff option function; see an example in the package unit tests for information.

When multiple addresses are passed in configuration, the package will use them in a round-robin fashion,
and will keep track of live and dead nodes. The status of dead nodes is checked periodically.

To customize the node selection behaviour, provide a Selector implementation in the configuration.
To replace the connection pool entirely, provide a custom ConnectionPool implementation via
the ConnectionPoolFunc option.

The package defines the Logger interface for logging information about request and response.
It comes with several bundled loggers for logging in text and JSON.

Use the EnableDebugLogger option to enable the debugging logger for connection management.

Use the EnableMetrics option to enable metric collection and export.
*/
package estransport
