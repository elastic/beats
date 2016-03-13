package dns

import (
	"fmt"
)

// All dns protocol errors are defined here.

type Error interface {
	error
	ResponseError() string
}

type DNSError struct {
	Err string
}

func (e *DNSError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return e.Err
}

func (e *DNSError) ResponseError() string {
	return "Response: " + e.Error()
}

// Common
var (
	NonDnsMsg         = &DNSError{Err: "Message's data could not be decoded as DNS"}
	DuplicateQueryMsg = &DNSError{Err: "Another query with the same DNS ID from this client " +
		"was received so this query was closed without receiving a response"}
	NoResponse       = &DNSError{Err: "No response to this query was received"}
	OrphanedResponse = &DNSError{Err: "Response: received without an associated Query"}
)

// EDNS
var (
	UdpPacketTooLarge  = &DNSError{Err: fmt.Sprintf("Non-EDNS packet has size greater than %d", MaxDnsPacketSize)}
	RespEdnsNoSupport  = &DNSError{Err: "Responder does not support EDNS"}
	RespEdnsUnexpected = &DNSError{Err: "Unexpected EDNS answer"}
)

// TCP
var (
	ZeroLengthMsg       = &DNSError{Err: "Message's length was set to zero"}
	UnexpectedLengthMsg = &DNSError{Err: "Unexpected message data length"}
	IncompleteMsg       = &DNSError{Err: "Message's data is incomplete"}
)
