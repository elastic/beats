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
		{
			Line: `apiserver_request_count{client="Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36",code="200",contentType="",resource="elasticsearchclusters",verb="LIST"} 1`,
			Event: PromEvent{
				key:       "apiserver_request_count",
				value:     int64(1),
				labelHash: `client="Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36",code="200",contentType="",resource="elasticsearchclusters",verb="LIST"`,
				labels: common.MapStr{
					"client":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36",
					"code":        int64(200),
					"contentType": "",
					"resource":    "elasticsearchclusters",
					"verb":        "LIST",
				},
			},
		},
	}

	for _, test := range tests {
		event := NewPromEvent(test.Line)
		assert.Equal(t, event, test.Event)
	}
}
