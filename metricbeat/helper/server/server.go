package server

import "github.com/elastic/beats/libbeat/common"

type Meta common.MapStr

const (
	EventDataKey = "data"
)

// Server is an interface that can be used to implement servers which can accept data.
type Server interface {
	// Start is used to start the server at a well defined port.
	Start() error
	// Stop the server.
	Stop()
	// Get a channel of events.
	GetEvents() chan Event
}

// Event is an interface that can be used to get the event and event source related information.
type Event interface {
	// Get the raw bytes of the event.
	GetEvent() common.MapStr
	// Get any metadata associated with the data that was received. Ex: client IP for udp message,
	// request/response headers for HTTP call.
	GetMeta() Meta
}
