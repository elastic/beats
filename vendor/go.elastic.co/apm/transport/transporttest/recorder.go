// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package transporttest

import (
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"sync"

	"github.com/google/go-cmp/cmp"

	"go.elastic.co/apm"
	"go.elastic.co/apm/model"
)

// NewRecorderTracer returns a new apm.Tracer and
// RecorderTransport, which is set as the tracer's transport.
//
// DEPRECATED. Use apmtest.NewRecordingTracer instead.
func NewRecorderTracer() (*apm.Tracer, *RecorderTransport) {
	var transport RecorderTransport
	tracer, err := apm.NewTracerOptions(apm.TracerOptions{
		ServiceName: "transporttest",
		Transport:   &transport,
	})
	if err != nil {
		panic(err)
	}
	return tracer, &transport
}

// RecorderTransport implements transport.Transport, recording the
// streams sent. The streams can be retrieved using the Payloads
// method.
type RecorderTransport struct {
	mu       sync.Mutex
	metadata *metadata
	payloads Payloads
}

// ResetPayloads clears out any recorded payloads.
func (r *RecorderTransport) ResetPayloads() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.payloads = Payloads{}
}

// SendStream records the stream such that it can later be obtained via Payloads.
func (r *RecorderTransport) SendStream(ctx context.Context, stream io.Reader) error {
	return r.record(ctx, stream)
}

// SendProfile records the stream such that it can later be obtained via Payloads.
func (r *RecorderTransport) SendProfile(ctx context.Context, metadata io.Reader, profiles ...io.Reader) error {
	return r.recordProto(ctx, metadata, profiles)
}

// Metadata returns the metadata recorded by the transport. If metadata is yet to
// be received, this method will panic.
func (r *RecorderTransport) Metadata() (_ model.System, _ model.Process, _ model.Service, labels model.StringMap) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.metadata.System, r.metadata.Process, r.metadata.Service, r.metadata.Labels
}

// Payloads returns the payloads recorded by SendStream.
func (r *RecorderTransport) Payloads() Payloads {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.payloads
}

func (r *RecorderTransport) record(ctx context.Context, stream io.Reader) error {
	reader, err := zlib.NewReader(stream)
	if err != nil {
		if err == io.ErrUnexpectedEOF {
			if contextDone(ctx) {
				return ctx.Err()
			}
			// truly unexpected
		}
		panic(err)
	}
	decoder := json.NewDecoder(reader)

	// The first object of any request must be a metadata struct.
	var metadataPayload struct {
		Metadata metadata `json:"metadata"`
	}
	if err := decoder.Decode(&metadataPayload); err != nil {
		panic(err)
	}
	r.recordMetadata(&metadataPayload.Metadata)

	for {
		var payload struct {
			Error       *model.Error       `json:"error"`
			Metrics     *model.Metrics     `json:"metricset"`
			Span        *model.Span        `json:"span"`
			Transaction *model.Transaction `json:"transaction"`
		}
		err := decoder.Decode(&payload)
		if err == io.EOF || (err == io.ErrUnexpectedEOF && contextDone(ctx)) {
			break
		} else if err != nil {
			panic(err)
		}
		r.mu.Lock()
		switch {
		case payload.Error != nil:
			r.payloads.Errors = append(r.payloads.Errors, *payload.Error)
		case payload.Metrics != nil:
			r.payloads.Metrics = append(r.payloads.Metrics, *payload.Metrics)
		case payload.Span != nil:
			r.payloads.Spans = append(r.payloads.Spans, *payload.Span)
		case payload.Transaction != nil:
			r.payloads.Transactions = append(r.payloads.Transactions, *payload.Transaction)
		}
		r.mu.Unlock()
	}
	return nil
}

func (r *RecorderTransport) recordProto(ctx context.Context, metadataReader io.Reader, profileReaders []io.Reader) error {
	var metadata metadata
	if err := json.NewDecoder(metadataReader).Decode(&metadata); err != nil {
		panic(err)
	}
	r.recordMetadata(&metadata)

	r.mu.Lock()
	defer r.mu.Unlock()
	for _, profileReader := range profileReaders {
		data, err := ioutil.ReadAll(profileReader)
		if err != nil {
			panic(err)
		}
		r.payloads.Profiles = append(r.payloads.Profiles, data)
	}
	return nil
}

func (r *RecorderTransport) recordMetadata(m *metadata) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.metadata == nil {
		r.metadata = m
	} else {
		// Make sure the metadata doesn't change between requests.
		if diff := cmp.Diff(r.metadata, m); diff != "" {
			panic(fmt.Errorf("metadata changed\n%s", diff))
		}
	}
}

func contextDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// Payloads holds the recorded payloads.
type Payloads struct {
	Errors       []model.Error
	Metrics      []model.Metrics
	Spans        []model.Span
	Transactions []model.Transaction
	Profiles     [][]byte
}

// Len returns the number of recorded payloads.
func (p *Payloads) Len() int {
	return len(p.Transactions) + len(p.Errors) + len(p.Metrics)
}

type metadata struct {
	System  model.System    `json:"system"`
	Process model.Process   `json:"process"`
	Service model.Service   `json:"service"`
	Labels  model.StringMap `json:"labels,omitempty"`
}
