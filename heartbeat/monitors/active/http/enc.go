package http

import (
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"strings"
)

type contentEncoder interface {
	AddHeaders(*http.Header)
	Encode(to io.Writer, from io.Reader) error
}

type nilEncoder struct{}

type gzipEncoder struct {
	gz *gzip.Writer
}

var plainEncoder = nilEncoder{}

func getContentEncoder(name string, level int) (contentEncoder, error) {
	name = strings.ToLower(name)
	if name == "gzip" {
		return newGZIPEncoder(level)
	}
	if name != "" {
		return nil, errors.New("invalid content encoder")
	}

	return plainEncoder, nil
}

func (nilEncoder) AddHeaders(_ *http.Header) {}

func (nilEncoder) Encode(to io.Writer, from io.Reader) error {
	_, err := io.Copy(to, from)
	return err
}

func newGZIPEncoder(level int) (*gzipEncoder, error) {
	w, err := gzip.NewWriterLevel(nil, level)
	if err != nil {
		return nil, err
	}

	return &gzipEncoder{w}, nil
}

func (e *gzipEncoder) AddHeaders(h *http.Header) {
	h.Add("Content-Type", "application/json; charset=UTF-8")
	h.Add("Content-Encoding", "gzip")
}

func (e *gzipEncoder) Encode(to io.Writer, from io.Reader) error {
	e.gz.Reset(to)
	if _, err := io.Copy(e.gz, from); err != nil {
		return err
	}
	if err := e.gz.Flush(); err != nil {
		return err
	}
	return nil
}
