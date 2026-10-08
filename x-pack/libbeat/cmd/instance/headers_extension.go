// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package instance

import (
	"net/http"
	"sort"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensionauth"

	"github.com/elastic/elastic-agent-libs/logp"
)

// headersSetterType is the component type of the OpenTelemetry Collector
// contrib headers_setter extension
// (github.com/open-telemetry/opentelemetry-collector-contrib/extension/headerssetterextension).
//
// We match on the type string rather than importing the extension package:
// Beats does not depend on it, and all we need from it is the
// extensionauth.HTTPClient interface it implements, which lives in the core
// collector module we already depend on.
const headersSetterType = "headers_setter"

// httpTransportWrapperFromExtensions looks for a headers_setter extension on
// the collector host and returns a function that wraps an http.RoundTripper
// with the extension's RoundTripper. The result is stored in
// beat.Info.HTTPTransportWrapper and applied by inputs and modules to the
// HTTP clients they create, so the headers configured on the extension (for
// #53390, the User-Agent Elastic Agent wants us to send) end up on their
// outgoing requests.
//
// This is the extension used the way it is designed to be used: as a client
// authenticator whose RoundTripper is layered onto an outgoing client. The
// difference from the usual `auth.authenticator` wiring is that the receiver
// config does not reference the extension; we discover it on the host by
// type, like the diagnostics and action extensions above in Start.
//
// Limitations, in rough order of how much they matter:
//
//  1. Only clients built with elastic-agent-libs/transport/httpcommon (or that
//     expose their http.RoundTripper) can be wrapped. That covers metricbeat's
//     helper.HTTP and the CEL, httpjson and streaming/crowdstrike inputs. It
//     does not cover:
//     - gcppubsub, which speaks gRPC. The extension also implements
//     extensionauth.GRPCClient, but gRPC metadata from PerRPCCredentials is
//     appended after grpc-go's own "user-agent" header field without the
//     reserved-header filter, so the request would carry two user-agent
//     fields. The input keeps using option.WithUserAgent (see the comment
//     in x-pack/filebeat/input/gcppubsub/input.go).
//     - o365audit, which sends through autorest.Send and autorest's
//     package-global http.Client. Wrapping it would mean building our own
//     client for the poller, which is out of scope for this change.
//     - The streaming input's websocket type, whose gorilla Dialer has no
//     RoundTripper and sends no User-Agent at all today.
//     - The `useragent` variable exposed to CEL programs (cel and streaming
//     inputs) and the httpjson `userAgent` template function, which are
//     values handed to user code, not transports.
//  2. Where the extension sits in each client's RoundTripper chain decides
//     who wins. Beats' own User-Agent RoundTrippers only fill the header if
//     it is missing, so the extension is wrapped outermost (runs first) and
//     both `insert` and `upsert` actions take effect. The exception is the
//     crowdstrike stream, whose userAgentTransport overwrites
//     unconditionally; there the extension is wrapped inside it so `upsert`
//     still wins. httpjson sets User-Agent on the *http.Request before any
//     RoundTripper runs, so only `upsert` changes it there.
//  3. Discovery is by component type. If the collector has more than one
//     headers_setter extension we cannot know which is meant for us; we pick
//     the first by ID and log the ambiguity.
//  4. Only `value` and `value_file` header sources are useful here.
//     `from_context` and `from_attribute` read the metadata of an incoming
//     collector request; our outgoing requests carry no such context, so
//     they resolve to default_value (or empty).
//  5. beat.Info.Version is still the beat's own build version in receiver
//     mode (the second half of #53390); this only addresses the headers.
func httpTransportWrapperFromExtensions(extensions map[component.ID]component.Component, log *logp.Logger) func(http.RoundTripper) http.RoundTripper {
	ids := make([]component.ID, 0, 1)
	for id := range extensions {
		if id.Type().String() == headersSetterType {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	// Sort so the choice is deterministic across restarts when there are
	// several candidates.
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	if len(ids) > 1 {
		log.Warnf("found %d headers_setter extensions (%v); the receiver cannot tell which one is intended for it and will use %s",
			len(ids), ids, ids[0])
	}
	id := ids[0]

	httpClient, ok := extensions[id].(extensionauth.HTTPClient)
	if !ok {
		log.Warnf("extension %s does not implement extensionauth.HTTPClient; outgoing requests will not use its headers", id)
		return nil
	}

	log.Infof("outgoing HTTP requests will carry the headers configured on extension %s", id)
	return func(base http.RoundTripper) http.RoundTripper {
		if base == nil {
			base = http.DefaultTransport
		}
		rt, err := httpClient.RoundTripper(base)
		if err != nil {
			// RoundTripper only fails when additional_auth is configured and
			// that extension is missing or broken. Falling back to the bare
			// transport keeps the input working without the headers.
			log.Warnf("extension %s: could not build RoundTripper, continuing without it: %v", id, err)
			return base
		}
		return rt
	}
}
