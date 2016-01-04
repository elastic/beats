package eventlogging

import "fmt"

// SID represents the Windows Security Identifier for an account.
type SID struct {
	Name    string
	Domain  string
	SIDType SIDType
}

// String returns string representation of SID.
func (a SID) String() string {
	return fmt.Sprintf("SID Name[%s] Domain[%s] SIDType[%s]",
		a.Name, a.Domain, a.SIDType)
}

// SIDType identifies the type of a security identifier (SID).
type SIDType uint32

// SIDType values.
const (
	// Do not reorder.
	SidTypeUser SIDType = 1 + iota
	SidTypeGroup
	SidTypeDomain
	SidTypeAlias
	SidTypeWellKnownGroup
	SidTypeDeletedAccount
	SidTypeInvalid
	SidTypeUnknown
	SidTypeComputer
	SidTypeLabel
)

// Mapping of SID types to their string representations.
var sidTypeToString = map[SIDType]string{
	SidTypeUser:           "User",
	SidTypeGroup:          "Group",
	SidTypeDomain:         "Domain",
	SidTypeAlias:          "Alias",
	SidTypeWellKnownGroup: "Well Known Group",
	SidTypeDeletedAccount: "Deleted Account",
	SidTypeInvalid:        "Invalid",
	SidTypeUnknown:        "Unknown",
	SidTypeComputer:       "Computer",
	SidTypeLabel:          "Label",
}

// String returns string representation of SIDType.
func (st SIDType) String() string {
	return sidTypeToString[st]
}

// MessageFiles contains handles to event message files associated with an
// event log source.
type MessageFiles struct {
	SourceName string
	Err        error
	Handles    []FileHandle
}

// FileHandle contains the handle to a single Windows message file.
type FileHandle struct {
	File   string  // Fully-qualified path to the event message file.
	Handle uintptr // Handle to the loaded event message file.
	Err    error   // Error that occurred while loading Handle.
}

// InsufficientBufferError indicates the buffer passed to a system call is too
// small.
type InsufficientBufferError struct {
	Cause        error
	RequiredSize int // Size of the buffer that is required.
}

func (e InsufficientBufferError) Error() string {
	return e.Cause.Error()
}
