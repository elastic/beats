package dns

import (
	"fmt"
)

// All dns protocol errors are defined here.

type DNSError struct {
	message string
}

func (e *DNSError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return e.message
}

func (e *DNSError) ResponseError() string {
	return "Response: " + e.Error()
}

// Common
var (
	nonDNSMsg         = &DNSError{message: "Message's data could not be decoded as DNS"}
	duplicateQueryMsg = &DNSError{message: "Another query with the same DNS ID from this client " +
		"was received so this query was closed without receiving a response"}
	noResponse       = &DNSError{message: "No response to this query was received"}
	orphanedResponse = &DNSError{message: "Response: received without an associated Query"}
)

// EDNS
var (
	udpPacketTooLarge  = &DNSError{message: fmt.Sprintf("Non-EDNS packet has size greater than %d", maxDNSPacketSize)}
	respEdnsNoSupport  = &DNSError{message: "Responder does not support EDNS"}
	respEdnsUnexpected = &DNSError{message: "Unexpected EDNS answer"}
)

// TCP
var (
	zeroLengthMsg       = &DNSError{message: "Message's length was set to zero"}
	unexpectedLengthMsg = &DNSError{message: "Unexpected message data length"}
	incompleteMsg       = &DNSError{message: "Message's data is incomplete"}
)
