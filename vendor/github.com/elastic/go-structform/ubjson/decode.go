package ubjson

import (
	"io"

	structform "github.com/elastic/go-structform"
)

type Decoder struct {
	p Parser

	buffer  []byte
	buffer0 []byte
	in      io.Reader
}

func NewDecoder(in io.Reader, buffer int, vs structform.Visitor) *Decoder {
	dec := &Decoder{
		buffer0: make([]byte, buffer),
		in:      in,
	}
	dec.p.init(vs)
	return dec
}

func NewBytesDecoder(b []byte, vs structform.Visitor) *Decoder {
	dec := &Decoder{
		buffer:  b,
		buffer0: b[:0],
		in:      nil,
	}
	dec.p.init(vs)
	return dec
}

func (dec *Decoder) Next() error {
	var (
		n        int
		err      error
		reported bool
	)

	for !reported {
		if len(dec.buffer) == 0 {
			if dec.in == nil {
				if err := dec.p.finalize(); err != nil {
					return err
				}
				return io.EOF
			}

			n, err := dec.in.Read(dec.buffer0)
			dec.buffer = dec.buffer0[:n]
			if n == 0 && err != nil {
				return err
			}
		}

		n, reported, err = dec.p.feedUntil(dec.buffer)
		if err != nil {
			return err
		}

		dec.buffer = dec.buffer[n:]
		if reported {
			return nil
		}
	}

	return nil
}
