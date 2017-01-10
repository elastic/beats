// +build !integration

package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestDecodeLine(t *testing.T) {
	tests := []struct {
		Line  string
		Event PromEvent
	}{
		{
			Line: `http_request_duration_microseconds{handler="query",quantile="0.99"} 17`,
			Event: PromEvent{
				key:       "http_request_duration_microseconds",
				value:     int64(17),
				labelHash: `handler="query",quantile="0.99"`,
				labels: common.MapStr{
					"handler":  "query",
					"quantile": 0.99,
				},
			},
		},
		{
			Line: `http_request_duration_microseconds{handler="query",quantile="0.99"} NaN`,
			Event: PromEvent{
				key:       "http_request_duration_microseconds",
				value:     nil,
				labelHash: `handler="query",quantile="0.99"`,
				labels: common.MapStr{
					"handler":  "query",
					"quantile": 0.99,
				},
			},
		},
		{
			Line: `http_request_duration_microseconds{handler="query",quantile="0.99"} 13.2`,
			Event: PromEvent{
				key:       "http_request_duration_microseconds",
				value:     13.2,
				labelHash: `handler="query",quantile="0.99"`,
				labels: common.MapStr{
					"handler":  "query",
					"quantile": 0.99,
				},
			},
		},
	}

	for _, test := range tests {
		event := NewPromEvent(test.Line)
		assert.Equal(t, event, test.Event)
	}
}
