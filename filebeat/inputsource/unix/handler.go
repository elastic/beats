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

package unix

import (
	"bufio"
	"net"
	"time"

	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/filebeat/inputsource/common"
)

// SplitHandlerFactory allows creation of a ConnectionHandler that can do splitting of messages received on a TCP connection.
func SplitHandlerFactory(callback inputsource.NetworkFunc, splitFunc bufio.SplitFunc) common.HandlerFactory {
	return func(config common.ListenerConfig) common.ConnectionHandler {
		return newSplitHandler(
			callback,
			splitFunc,
			uint64(config.MaxMessageSize),
			config.Timeout,
		)
	}
}

// splitHandler is a TCP handler that has splitting capabilities.
type splitHandler struct {
	common.ConnectionHandler
	callback inputsource.NetworkFunc
	metadata inputsource.NetworkMetadata
}

// newSplitHandler allows creation of a TCP client that has splitting capabilities.
func newSplitHandler(
	callback inputsource.NetworkFunc,
	splitFunc bufio.SplitFunc,
	maxReadMessage uint64,
	timeout time.Duration,
) common.ConnectionHandler {
	handler := &splitHandler{
		callback: callback,
	}
	handler.ConnectionHandler = common.NewSplitHandler(
		common.FamilyUnix,
		handler.onStart,
		handler.onLine,
		splitFunc,
		maxReadMessage,
		timeout,
	)
	return handler
}

func (c *splitHandler) onStart(conn net.Conn) {
	c.metadata = inputsource.NetworkMetadata{}
}

func (c *splitHandler) onLine(data []byte) {
	c.callback(data, c.metadata)
}
