// Package loggregator provides clients to send data to the Loggregator v1 and
// v2 API.
//
// The v2 API distinguishes itself from the v1 API on three counts:
//
// 1) it uses gRPC,
// 2) it uses a streaming connection, and
// 3) it supports batching to improve performance.
//
// The code here provides a generic interface into the two APIs. Clients who
// prefer more fine grained control may generate their own code using the
// protobuf and gRPC service definitions found at:
// github.com/cloudfoundry/loggregator-api.
//
// Note that on account of the client using batching wherein multiple
// messages may be sent at once, there is no meaningful error return value
// available. Each of the methods below make a best-effort at message
// delivery. Even in the event of a failed send, the client will not block
// callers.
//
// In general, use IngressClient for communicating with Loggregator's v2 API.
// For Loggregator's v1 API, see v1/client.go.
package loggregator
