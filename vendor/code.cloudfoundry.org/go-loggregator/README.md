# go-loggregator
[![GoDoc][go-doc-badge]][go-doc] [![travis][travis-badge]][travis] [![slack.cloudfoundry.org][slack-badge]][loggregator-slack]

This is a golang client library for [Loggregator][loggregator].

## Versions

At present, Loggregator supports two API versions: v1 (UDP) and v2 (gRPC).
This library provides clients for both versions.

Note that this library is also versioned. Its versions have *no* relation to
the Loggregator API.

## Usage

This repository should be imported as:

`import loggregator "code.cloudfoundry.org/go-loggregator"`

## Examples

To build the examples, `cd` into the directory of the example and run `go build`

### V1 Ingress

Emits envelopes to metron using dropsonde.

### V2 Ingress

Emits envelopes to metron using the V2 loggregator-api.

Required Environment Variables:

* `CA_CERT_PATH`
* `CERT_PATH`
* `KEY_PATH`

### Runtime Stats

Emits information about the running Go proccess using a V2 ingress client.

Required Environment Variables:

* `CA_CERT_PATH`
* `CERT_PATH`
* `KEY_PATH`

### Envelope Stream Connector

Reads envelopes from the Loggregator API (e.g. Reverse Log Proxy).

Required Environment Variables:

* `CA_CERT_PATH`
* `CERT_PATH`
* `KEY_PATH`
* `LOGS_API_ADDR`
* `SHARD_ID`

[slack-badge]:              https://slack.cloudfoundry.org/badge.svg
[loggregator-slack]:        https://cloudfoundry.slack.com/archives/loggregator
[loggregator]:              https://github.com/cloudfoundry/loggregator
[go-doc-badge]:             https://godoc.org/code.cloudfoundry.org/go-loggregator?status.svg
[go-doc]:                   https://godoc.org/code.cloudfoundry.org/go-loggregator
[travis-badge]:             https://travis-ci.org/cloudfoundry/go-loggregator.svg?branch=master
[travis]:                   https://travis-ci.org/cloudfoundry/go-loggregator?branch=master
