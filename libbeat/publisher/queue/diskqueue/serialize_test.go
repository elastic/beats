package diskqueue

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/stretchr/testify/assert"
)

// A test to make sure serialization works correctly on multi-byte characters.
func TestSerializeMultiByte(t *testing.T) {
	asciiOnly := "{\"name\": \"Momotaro\"}"
	multiBytes := "{\"name\": \"桃太郎\"}"

	encoder := newEventEncoder()
	event := publisher.Event{
		Content: beat.Event{
			Fields: common.MapStr{
				"ascii_only":  asciiOnly,
				"multi_bytes": multiBytes,
			},
		},
	}
	serialized, err := encoder.encode(&event)
	if err != nil {
		t.Fatalf("Couldn't encode event: %v", err)
	}

	// Use decoder to decode the serialized bytes.
	decoder := newEventDecoder()
	buf := decoder.Buffer(len(serialized))
	copy(buf, serialized)
	decoded, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Couldn't decode serialized data: %v", err)
	}

	decodedAsciiOnly, _ := decoded.Content.Fields.GetValue("ascii_only")
	assert.Equal(t, asciiOnly, decodedAsciiOnly)

	decodedMultiBytes, _ := decoded.Content.Fields.GetValue("multi_bytes")
	assert.Equal(t, multiBytes, decodedMultiBytes)
}
