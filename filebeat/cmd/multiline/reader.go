package multiline

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/elastic/beats/filebeat/harvester/encoding"
	"github.com/elastic/beats/filebeat/harvester/reader"
)

type concatReader struct {
	current io.ReadCloser
	idx     int
	paths   []string
}

func createReader(paths []string) (io.ReadCloser, error) {
	switch len(paths) {
	case 0:
		return ioutil.NopCloser(os.Stdin), nil
	case 1:
		return os.Open(paths[0])
	}
	return newConcatReader(paths)
}

func createPipeline(config *prospectorConfig, in io.Reader) (reader.Reader, error) {
	var (
		r   reader.Reader
		err error
	)

	ef, ok := encoding.FindEncoding(config.Encoding)
	if !ok || ef == nil {
		return nil, fmt.Errorf("unknown encoding('%v')", config.Encoding)
	}

	enc, err := ef(in)
	if err != nil {
		return nil, err
	}

	r, err = reader.NewEncode(in, enc, 8*1024)
	if err != nil {
		return nil, err
	}

	r = reader.NewStripNewline(r)

	r, err = reader.NewMultiline(r, "\n", config.MaxBytes, &config.Multiline)
	if err != nil {
		return nil, err
	}

	return reader.NewLimit(r, config.MaxBytes), nil
}

func newConcatReader(paths []string) (*concatReader, error) {
	init, err := os.Open(paths[0])
	if err != nil {
		return nil, err
	}

	return &concatReader{
		current: init,
		idx:     0,
		paths:   paths,
	}, nil
}

func (r *concatReader) Close() (err error) {
	if r.current != nil {
		err = r.current.Close()
		r.current = nil
	}
	return
}

func (r *concatReader) Read(b []byte) (int, error) {
	N := 0
	for len(b) > 0 {
		n, err := r.current.Read(b)
		N += n
		if err == nil || err != io.EOF {
			return N, nil
		}

		b = b[n:]

		// close old file
		r.current.Close()
		r.current = nil

		// advance to next file
		r.idx++
		if r.idx >= len(r.paths) {
			return n, io.EOF
		}
		next, err := os.Open(r.paths[r.idx])
		if err != nil {
			return n, err
		}
		r.current = next
	}
	return N, nil
}
