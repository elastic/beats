package dns

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

// Messages
var (
	NonDnsMsg           = &DNSError{Err: "Message's data could not be decoded as DNS"}
	ZeroLengthMsg       = &DNSError{Err: "Message's length was set to zero"}
	UnexpectedLengthMsg = &DNSError{Err: "Unexpected message data length"}
	DuplicateQueryMsg   = &DNSError{Err: "Another query with the same DNS ID from this client " +
		"was received so this query was closed without receiving a response"}
	IncompleteMsg = &DNSError{Err: "Message's data is incomplete"}
	NoResponse    = &DNSError{Err: "No response to this query was received"}
)

// TCP responses
var (
	OrphanedResponse = &DNSError{Err: "Response: received without an associated Query"}
)
