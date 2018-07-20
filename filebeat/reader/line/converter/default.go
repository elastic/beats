package converter

import (
	"fmt"

	"golang.org/x/text/transform"

	"github.com/elastic/beats/libbeat/common/streambuf"
)

type DefaultConverter struct {
	transformer transform.Transformer
	buf         *streambuf.Buffer
	symlen      []uint8

	bufferSize int
}

func init() {
	err := register("default", newDefaultConverter)
	if err != nil {
		panic(err)
	}
}

func newDefaultConverter(t transform.Transformer, size int) (Converter, error) {
	return &DefaultConverter{
		transformer: t,
		buf:         streambuf.New(nil),
		symlen:      make([]uint8, size),
		bufferSize:  size,
	}, nil
}

// msgSize returns the size of the encoded message on the disk
func (c *DefaultConverter) MsgSize(symlen []uint8, size int) (int, []uint8, error) {
	n := 0
	for size > 0 {
		if len(symlen) <= n {
			return 0, symlen, fmt.Errorf("error calculating size: too short symlen")
		}

		size -= int(symlen[n])
		n++
	}

	symlen = symlen[n:]

	return n, symlen, nil
}

func (c *DefaultConverter) GetSymLen() []uint8 {
	s := c.symlen
	c.symlen = []uint8{}
	return s
}

// conv converts encoded bytes into UTF-8 and produces a symlen array which
// records the size of the encoded bytes and its converted size
func (c *DefaultConverter) Convert(in []byte, out []byte) (int, int, error) {
	var err error
	nProcessed := 0
	decodedChar := make([]byte, 64)
	c.symlen = make([]uint8, len(in))

	i := 0
	srcLen := len(in)
	for i < srcLen {
		j := i + 1

		for j <= srcLen {
			nDst, nSrc, err := c.transformer.Transform(decodedChar, in[i:j], false)
			if err != nil {
				// if no char is decoded, try increasing the input buffer
				if err == transform.ErrShortSrc {
					j++

					// if the buffer size cannot be increased, return what's been decoded and an error
					if srcLen < j {
						n, _ := c.copyToOut(out)
						c.symlen = c.symlen[:nProcessed]
						return n, nProcessed, err
					}
				}
				err = nil
			}

			// move in the symlen buffer if no char is decoded
			if nDst == 0 && nSrc == 0 {
				nProcessed++
				continue
			}

			c.symlen[nProcessed] = uint8(nDst)
			c.buf.Write(decodedChar[:nDst])
			nProcessed++
			break
		}
		i = j
	}

	n, err := c.copyToOut(out)
	return n, nProcessed, err
}

func (c *DefaultConverter) Collect(out []byte) (int, error) {
	if c.buf.Len() > 0 {
		return c.copyToOut(out)
	}
	return 0, nil
}

func (c *DefaultConverter) copyToOut(out []byte) (int, error) {
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
