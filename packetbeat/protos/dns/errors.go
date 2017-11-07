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
