package reader

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
)

// Message represents a reader event with timestamp, content and actual number
// of bytes read from input before decoding.
type Message struct {
	Ts      time.Time     // timestamp the content was read
	Content []byte        // actual content read
	Bytes   int           // total number of bytes read to generate the message
	Fields  common.MapStr // optional fields that can be added by reader
}

// Reader is the interface that wraps the basic Next method for
// getting a new message.
// Next returns the message being read or and error. EOF is returned
// if reader will not return any new message on subsequent calls.
type Reader interface {
	Next() (Message, error)
}
