package hepa

import (
	"bufio"
	"bytes"
	"io"

	"github.com/markbates/pkger/internal/takeon/github.com/markbates/hepa/filters"
)

type Purifier struct {
	parent *Purifier
	filter Filter
}

func (p Purifier) Filter(b []byte) ([]byte, error) {
	if p.filter == nil {
		p.filter = filters.Home()
	}
	b, err := p.filter.Filter(b)
	if err != nil {
		return b, err
	}
	if p.parent != nil {
		return p.parent.Filter(b)
	}
	return b, nil
}

func (p Purifier) Clean(r io.Reader) ([]byte, error) {
	bb := &bytes.Buffer{}

	if p.filter == nil {
		if p.parent != nil {
			return p.parent.Clean(r)
		}
		_, err := io.Copy(bb, r)
		return bb.Bytes(), err
	}

	home := filters.Home()
	reader := bufio.NewReader(r)
	for {
		input, _, err := reader.ReadLine()
		if err != nil && err == io.EOF {
			break
		}
		input, err = p.Filter(input)
		if err != nil {
			return nil, err
		}
		input, err = home(input)
		if err != nil {
			return nil, err
		}
		bb.Write(input)
		// if len(input) > 0 {
		bb.Write([]byte("\n"))
		// }
	}

	return bb.Bytes(), nil
}

func New() Purifier {
	return Purifier{}
}
