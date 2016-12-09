package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

type RespCheck func(*http.Response) error

var (
	errBodyMismatch = errors.New("body mismatch")
	errStatus404    = errors.New("file not found")
)

func makeValidateResponse(config *checkConfig) RespCheck {
	var checks []RespCheck

	if config.Status > 0 {
		checks = append(checks, checkStatus(config.Status))
	} else {
		checks = append(checks, checkStatusNot404)
	}

	if len(config.RecvHeaders) > 0 {
		checks = append(checks, checkHeaders(config.RecvHeaders))
	}

	if len(config.RecvBody) > 0 {
		checks = append(checks, checkBody([]byte(config.RecvBody)))
	}

	switch len(checks) {
	case 0:
		return checkOK
	case 1:
		return checks[0]
	default:
		return checkAll(checks...)
	}
}

func checkOK(_ *http.Response) error { return nil }

// TODO: collect all errors into on error message.
func checkAll(checks ...RespCheck) RespCheck {
	return func(r *http.Response) error {
		for _, check := range checks {
			if err := check(r); err != nil {
				return err
			}
		}
		return nil
	}
}

func checkStatus(status uint16) RespCheck {
	return func(r *http.Response) error {
		if r.StatusCode == int(status) {
			return nil
		}
		return fmt.Errorf("received status code %v expecting %v", r.StatusCode, status)
	}
}

func checkStatusNot404(r *http.Response) error {
	if r.StatusCode == 404 {
		return errStatus404
	}
	return nil
}

func checkHeaders(headers map[string]string) RespCheck {
	return func(r *http.Response) error {
		for k, v := range headers {
			value := r.Header.Get(k)
			if v != value {
				return fmt.Errorf("header %v is '%v' expecting '%v' ", k, value, v)
			}
		}
		return nil
	}
}

func checkBody(body []byte) RespCheck {
	return func(r *http.Response) error {
		// read up to len(body)+1 bytes for comparing content to be equal
		in := io.LimitReader(r.Body, int64(len(body))+1)
		content, err := ioutil.ReadAll(in)
		if err != nil {
			return err
		}

		if !bytes.Equal(body, content) {
			return errBodyMismatch
		}
		return nil
	}
}
