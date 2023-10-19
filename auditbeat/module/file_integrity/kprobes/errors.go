package kprobes

import "errors"

var (
	ErrVerifyOverlappingEvents = errors.New("probe overlapping events")
	ErrVerifyMissingEvents     = errors.New("probe missing events")
	ErrVerifyUnexpectedEvent   = errors.New("received an event that was not expected")
	ErrSymbolNotFound          = errors.New("symbol not found")
	ErrAckTimeout              = errors.New("timeout")
)
