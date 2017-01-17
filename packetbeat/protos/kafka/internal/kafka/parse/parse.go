package parse

import (
	"github.com/elastic/beats/libbeat/common/streambuf"

	"github.com/elastic/beats/packetbeat/protos/kafka/internal/kafka"
)

type Parser struct {
	buffer    streambuf.Buffer
	onMessage func(m kafka.RawMessage) error
}

func NewParser(cb func(kafka.RawMessage) error, data []byte) *Parser {
	return &Parser{
		buffer:    *streambuf.New(data),
		onMessage: cb,
	}
}

// Feed passes more bytes to the parser initiating another parse step
func (p *Parser) Feed(b []byte) error {
	if err := p.Append(b); err != nil {
		return err
	}

	return p.StepAll()
}

func (p *Parser) Append(b []byte) error {
	return p.buffer.Append(b)
}

func (p *Parser) Step() error {
	pl := p.next()
	if pl != nil {
		p.onMessage(kafka.RawMessage{pl})
	}

	return nil
}

func (p *Parser) StepAll() error {
	for {
		pl := p.next()
		if pl == nil {
			break
		}

		p.onMessage(kafka.RawMessage{pl})
	}
	return nil
}

func (p *Parser) next() []byte {
	sz := p.nextSize()
	if sz < 0 {
		return nil
	}

	p.buffer.Advance(4)
	if sz == 0 {
		return nil
	}

	payload, _ := p.buffer.Collect(sz)
	p.buffer.Reset()
	return payload
}

// Size returns number of bytes currently buffered
func (p *Parser) Size() int {
	return p.buffer.Len()
}

// MessageBuffered returns true if a kafka message is available for parsing from buffer
func (p *Parser) MessageBuffered() bool {
	return p.nextSize() >= 0
}

func (p *Parser) nextSize() int {
	count, err := p.buffer.ReadNetUint32At(0)
	if err != nil {
		return -1
	}
	return int(count)
}
