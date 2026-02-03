// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package akamai implements an input for collecting security events from
// the Akamai SIEM API (https://techdocs.akamai.com/siem-integration/reference/api).
//
// The input uses EdgeGrid authentication to securely communicate with the Akamai API
// and supports:
//   - Worker-based parallel processing for scalability
//   - Cursor-based state management for reliable event collection
//   - Automatic pagination and offset tracking
//   - Recovery mode for handling timestamp-based gaps
//   - Rate limiting and retry with exponential backoff
//   - Comprehensive metrics and logging
//
// # Configuration
//
// Example configuration:
//
//	filebeat.inputs:
//	  - type: akamai
//	    enabled: true
//	    api_host: https://akzz-XXXXXXXX.luna.akamaiapis.net
//	    config_ids: "12345;67890"
//	    client_token: "akab-xxx"
//	    client_secret: "xxx"
//	    access_token: "akab-xxx"
//	    interval: 1m
//	    initial_interval: 12h
//	    event_limit: 10000
//	    number_of_workers: 3
//
// # Authentication
//
// The input uses Akamai EdgeGrid authentication (EG1-HMAC-SHA256), which requires:
//   - client_token: Client token from Akamai credentials
//   - client_secret: Client secret from Akamai credentials
//   - access_token: Access token from Akamai authorizations
//
// # Metrics
//
// The input exposes the following metrics:
//   - akamai_requests_total: Total number of API requests made
//   - akamai_events_received_total: Total number of events received
//   - akamai_events_published_total: Total number of events published
//   - akamai_batches_received_total: Total number of batches received
//   - akamai_batches_published_total: Total number of batches published
//   - akamai_errors_total: Total number of errors encountered
//   - akamai_worker_utilization: Current worker utilization (0-1)
//   - akamai_request_processing_time: Histogram of request processing times
//   - akamai_batch_processing_time: Histogram of batch processing times
package akamai
