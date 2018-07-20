package converter

import (
	"golang.org/x/text/transform"

	"github.com/elastic/beats/libbeat/common/streambuf"
)

func init() {
	err := register("plain", newPlainConverter)
	if err != nil {
		panic(err)
	}
}

type PlainConverter struct {
	transformer transform.Transformer
	buf         *streambuf.Buffer

	bufferSize int
}

func newPlainConverter(t transform.Transformer, size int) (Converter, error) {
	return &PlainConverter{
		transformer: t,
		buf:         streambuf.New(nil),
		bufferSize:  size,
	}, nil
}

func (c *PlainConverter) Convert(in, out []byte) (int, int, error) {
	nDst, nSrc, err := c.transformer.Transform(out, in, false)
	if err != nil {
		if err == transform.ErrShortSrc {
			n, _ := c.copyToOut(out)
			return n, nSrc, err
		}
	}
	err = nil

	return nDst, nSrc, err
}

func (c *PlainConverter) MsgSize(symlen []uint8, size int) (int, []uint8, error) {
	return size, nil, nil
}

func (c *PlainConverter) GetSymLen() []uint8 {
	return nil
}

func (c *PlainConverter) Collect(out []byte) (int, error) {
	if c.buf.Len() > 0 {
		return c.copyToOut(out)
	}
	return 0, nil
}

func (c *PlainConverter) copyToOut(out []byte) (int, error) {
	until := len(out)
	if c.buf.Len() < until {
		until = c.buf.Len()
	}
	b, err := c.buf.Collect(until)
	if err != nil {
		return 0, err
	}
	c.buf.Reset()
	return copy(out, b), nil
}
