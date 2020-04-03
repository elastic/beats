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

package common

import (
	"bufio"
	"net"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/logp"
)

// OnStartFunc defines callback executed when a connection is initialized.
type OnStartFunc = func(conn net.Conn)

// OnLineFunc defines callback executed when a line is read from the split handler.
type OnLineFunc = func(data []byte)

// SplitHandler is a TCP client that has splitting capabilities.
type SplitHandler struct {
	onStart        OnStartFunc
	onLine         OnLineFunc
	splitFunc      bufio.SplitFunc
	maxMessageSize uint64
	timeout        time.Duration
	family         Family
}

// NewSplitHandler allows creation of a TCP client that has splitting capabilities.
func NewSplitHandler(
	family Family,
	onStart OnStartFunc,
	onLine OnLineFunc,
	splitFunc bufio.SplitFunc,
	maxReadMessage uint64,
	timeout time.Duration,
) ConnectionHandler {
	return &SplitHandler{
		onStart:        onStart,
		onLine:         onLine,
		splitFunc:      splitFunc,
		maxMessageSize: maxReadMessage,
		timeout:        timeout,
		family:         family,
	}
}

// Handle takes a connection as input and processes data received on it.
func (c *SplitHandler) Handle(closer CloseRef, conn net.Conn) error {
	c.onStart(conn)

	var log *logp.Logger
	if c.family == FamilyUnix {
		// unix sockets have an empty `RemoteAddr` value, so no need to capture it
		log = logp.NewLogger("split_client")
	} else {
		log = logp.NewLogger("split_client").With("remote_addr", conn.RemoteAddr().String())
	}

	r := NewResetableLimitedReader(NewDeadlineReader(conn, c.timeout), c.maxMessageSize)
	buf := bufio.NewReader(r)
	scanner := bufio.NewScanner(buf)
	scanner.Split(c.splitFunc)
	//16 is ratio of MaxScanTokenSize/startBufSize
	buffer := make([]byte, c.maxMessageSize/16)
	scanner.Buffer(buffer, int(c.maxMessageSize))
	for {
		select {
		case <-closer.Done():
			break
		default:
		}

		// Ensure that if the Conn is already closed then dont attempt to scan again
		if closer.Err() == ErrClosed {
			break
		}

		if !scanner.Scan() {
			break
		}

		err := scanner.Err()
		if err != nil {
			// This is a user defined limit and we should notify the user.
			if IsMaxReadBufferErr(err) {
				log.Errorw("split_client error", "error", err)
			}
			return errors.Wrap(err, string(c.family)+" split_client error")
		}
		r.Reset()
		c.onLine(scanner.Bytes())
	}

	// We are out of the scanner, either we reached EOF or another fatal error occurred.
	// like we failed to complete the TLS handshake or we are missing the splitHandler certificate when
	// mutual auth is on, which is the default.
	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
