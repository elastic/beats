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

package dns

import (
	"fmt"
)

// All dns protocol errors are defined here.

type dnsError struct {
	message string
}

func (e *dnsError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return e.message
}

func (e *dnsError) responseError() string {
	return "Response: " + e.Error()
}

// Common
var (
	nonDNSMsg         = &dnsError{message: "Message's data could not be decoded as DNS"}
	duplicateQueryMsg = &dnsError{message: "Another query with the same DNS ID from this client " +
		"was received so this query was closed without receiving a response"}
	noResponse       = &dnsError{message: "No response to this query was received"}
	orphanedResponse = &dnsError{message: "Response: received without an associated Query"}
)

// EDNS
var (
	udpPacketTooLarge  = &dnsError{message: fmt.Sprintf("Non-EDNS packet has size greater than %d", maxDNSPacketSize)}
	respEdnsNoSupport  = &dnsError{message: "Responder does not support EDNS"}
	respEdnsUnexpected = &dnsError{message: "Unexpected EDNS answer"}
)

// TCP
var (
	zeroLengthMsg       = &dnsError{message: "Message's length was set to zero"}
	unexpectedLengthMsg = &dnsError{message: "Unexpected message data length"}
	incompleteMsg       = &dnsError{message: "Message's data is incomplete"}
)
