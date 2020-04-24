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
	"context"
	"io"
	"io/ioutil"

	"go.elastic.co/apm/transport"
)

// Discard is a transport.Transport which discards
// all streams, and returns no errors.
var Discard transport.Transport = ErrorTransport{}

// ErrorTransport is a transport that returns the stored error
// for each method call.
type ErrorTransport struct {
	Error error
}

// SendStream discards the stream and returns t.Error.
func (t ErrorTransport) SendStream(ctx context.Context, r io.Reader) error {
	errc := make(chan error, 1)
	go func() {
		_, err := io.Copy(ioutil.Discard, r)
		errc <- err
	}()
	select {
	case err := <-errc:
		if err != nil {
			return err
		}
		return t.Error
	case <-ctx.Done():
		return ctx.Err()
	}
}
