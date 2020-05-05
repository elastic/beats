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

package apm

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"sync"
	"unicode/utf8"

	"go.elastic.co/apm/internal/apmstrings"
	"go.elastic.co/apm/model"
)

// CaptureBodyMode holds a value indicating how a tracer should capture
// HTTP request bodies: for transactions, for errors, for both, or neither.
type CaptureBodyMode int

const (
	// CaptureBodyOff disables capturing of HTTP request bodies. This is
	// the default mode.
	CaptureBodyOff CaptureBodyMode = 0

	// CaptureBodyErrors captures HTTP request bodies for only errors.
	CaptureBodyErrors CaptureBodyMode = 1

	// CaptureBodyTransactions captures HTTP request bodies for only
	// transactions.
	CaptureBodyTransactions CaptureBodyMode = 1 << 1

	// CaptureBodyAll captures HTTP request bodies for both transactions
	// and errors.
	CaptureBodyAll CaptureBodyMode = CaptureBodyErrors | CaptureBodyTransactions
)

var bodyCapturerPool = sync.Pool{
	New: func() interface{} {
		return &BodyCapturer{}
	},
}

// CaptureHTTPRequestBody replaces req.Body and returns a possibly nil
// BodyCapturer which can later be passed to Context.SetHTTPRequestBody
// for setting the request body in a transaction or error context. If the
// tracer is not configured to capture HTTP request bodies, then req.Body
// is left alone and nil is returned.
//
// This must be called before the request body is read. The BodyCapturer's
// Discard method should be called after it is no longer needed, in order
// to recycle its memory.
func (t *Tracer) CaptureHTTPRequestBody(req *http.Request) *BodyCapturer {
	if req.Body == nil {
		return nil
	}
	captureBody := t.instrumentationConfig().captureBody
	if captureBody == CaptureBodyOff {
		return nil
	}

	bc := bodyCapturerPool.Get().(*BodyCapturer)
	bc.captureBody = captureBody
	bc.request = req
	bc.originalBody = req.Body
	bc.buffer.Reset()
	req.Body = bodyCapturerReadCloser{BodyCapturer: bc}
	return bc
}

// bodyCapturerReadCloser implements io.ReadCloser using the embedded BodyCapturer.
type bodyCapturerReadCloser struct {
	*BodyCapturer
}

// Close closes the original body.
func (bc bodyCapturerReadCloser) Close() error {
	return bc.originalBody.Close()
}

// Read reads from the original body, copying into bc.buffer.
func (bc bodyCapturerReadCloser) Read(p []byte) (int, error) {
	n, err := bc.originalBody.Read(p)
	if n > 0 {
		bc.buffer.Write(p[:n])
	}
	return n, err
}

// BodyCapturer is returned by Tracer.CaptureHTTPRequestBody to later be
// passed to Context.SetHTTPRequestBody.
//
// Calling Context.SetHTTPRequestBody will reset req.Body to its original
// value, and invalidates the BodyCapturer.
type BodyCapturer struct {
	captureBody CaptureBodyMode

	readbuf      [bytes.MinRead]byte
	buffer       limitedBuffer
	request      *http.Request
	originalBody io.ReadCloser
}

// Discard discards the body capturer: the original request body is
// replaced, and the body capturer is returned to a pool for reuse.
// The BodyCapturer must not be used after calling this.
//
// Discard has no effect if bc is nil.
func (bc *BodyCapturer) Discard() {
	if bc == nil {
		return
	}
	bc.request.Body = bc.originalBody
	bodyCapturerPool.Put(bc)
}

func (bc *BodyCapturer) setContext(out *model.RequestBody) bool {
	if bc.request.PostForm != nil {
		// We must copy the map in case we need to
		// sanitize the values. Ideally we should only
		// copy if sanitization is necessary, but body
		// capture shouldn't typically be enabled so
		// we don't currently optimize this.
		postForm := make(url.Values, len(bc.request.PostForm))
		for k, v := range bc.request.PostForm {
			vcopy := make([]string, len(v))
			for i := range vcopy {
				vcopy[i] = truncateString(v[i])
			}
			postForm[k] = vcopy
		}
		out.Form = postForm
		return true
	}

	body, n := apmstrings.Truncate(bc.buffer.String(), stringLengthLimit)
	if n == stringLengthLimit {
		// There is at least enough data in the buffer
		// to hit the string length limit, so we don't
		// need to read from bc.originalBody as well.
		out.Raw = body
		return true
	}

	// Read the remaining body, limiting to the maximum number of bytes
	// that could make up the truncation limit. We ignore any errors here,
	// and just return whatever we can.
	rem := utf8.UTFMax * (stringLengthLimit - n)
	for {
		buf := bc.readbuf[:]
		if rem < bytes.MinRead {
			buf = buf[:rem]
		}
		n, err := bc.originalBody.Read(buf)
		if n > 0 {
			bc.buffer.Write(buf[:n])
			rem -= n
		}
		if rem == 0 || err != nil {
			break
		}
	}
	body, _ = apmstrings.Truncate(bc.buffer.String(), stringLengthLimit)
	out.Raw = body
	return body != ""
}

type limitedBuffer struct {
	bytes.Buffer
}

func (b *limitedBuffer) Write(p []byte) (n int, err error) {
	rem := (stringLengthLimit * utf8.UTFMax) - b.Len()
	n = len(p)
	if n > rem {
		p = p[:rem]
	}
	written, err := b.Buffer.Write(p)
	if err != nil {
		n = written
	}
	return n, err
}
