package logstash

import (
	"github.com/elastic/beats/libbeat/outputs/codec/json"
	"github.com/elastic/beats/libbeat/publisher/beat"
)

func makeLogstashEventEncoder(index string) func(interface{}) ([]byte, error) {
	enc := json.New(false)
	return func(event interface{}) ([]byte, error) {
		return enc.Encode(index, event.(*beat.Event))
	}
}
