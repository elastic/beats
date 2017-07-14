package codec

import "github.com/elastic/beats/libbeat/publisher/beat"

type Codec interface {
	Encode(index string, event *beat.Event) ([]byte, error)
}
