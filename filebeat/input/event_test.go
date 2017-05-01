package input

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventToMapStr(t *testing.T) {
	// Test 'fields' is not present when it is nil.
	event := Event{}
	mapStr := event.ToMapStr()
	_, found := mapStr["fields"]
	assert.False(t, found)
}
