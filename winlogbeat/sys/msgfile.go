package sys

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
