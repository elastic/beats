package otel

import (
	"sync"

	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
)

type ConcurentEncoder struct {
	encoder stdoutmetric.Encoder
	lock    sync.Mutex
}

func NewConcurentEncoder(encoder stdoutmetric.Encoder) *ConcurentEncoder {
	return &ConcurentEncoder{
		encoder: encoder,
	}
}
func (ce ConcurentEncoder) Encode(v any) error {
	ce.lock.Lock()
	defer ce.lock.Unlock()
	return ce.encoder.Encode(v)
}
