package publisher

import (
	"testing"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/stretchr/testify/assert"
)

// Test that FilterEvent detects events that do not contain the required fields
// and returns error.
func TestFilterEvent(t *testing.T) {
	var testCases = []struct {
		f   func() common.MapStr
		err string
	}{
		{func() common.MapStr {
			return testEvent()
		}, ""},

		{func() common.MapStr {
			m := testEvent()
			m["@timestamp"] = time.Now()
			return m
		}, "Invalid '@timestamp'"},

		{func() common.MapStr {
			m := testEvent()
			delete(m, "@timestamp")
			return m
		}, "Missing '@timestamp'"},

		{func() common.MapStr {
			m := testEvent()
			delete(m, "type")
			return m
		}, "Missing 'type'"},

		{func() common.MapStr {
			m := testEvent()
			m["type"] = 123
			return m
		}, "Invalid 'type'"},
	}

	for _, test := range testCases {
		assert.Regexp(t, test.err, filterEvent(test.f()))
	}
}
