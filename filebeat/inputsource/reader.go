package inputsource

import (
	"io"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/reader"
)

type NetworkMessageReader struct {
	ready    bool
	raw      []byte
	metadata NetworkMetadata
}

func (r *NetworkMessageReader) SetData(raw []byte, metadata NetworkMetadata) {
	r.raw = raw
	r.metadata = metadata
	r.ready = true
}

func (p *NetworkMessageReader) Next() (reader.Message, error) {
	if !p.ready {
		return reader.Message{}, io.EOF
	}
	p.ready = false

	fields := common.MapStr{}
	return reader.Message{
		Content: p.raw,
		Bytes:   len(p.raw),
		Fields:  fields,
	}, nil
}

func (p *NetworkMessageReader) Close() error {
	p.ready = false
	return nil
}
