package http

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/elastic/beats/libbeat/common/match"
)

type RespCheck func(*http.Response) error

var (
	errBodyMismatch = errors.New("body mismatch")
)

func makeValidateResponse(config *responseParameters) RespCheck {
	var checks []RespCheck

	if config.Status > 0 {
		checks = append(checks, checkStatus(config.Status))
	} else {
		checks = append(checks, checkStatusOK)
	}

	if len(config.RecvHeaders) > 0 {
		checks = append(checks, checkHeaders(config.RecvHeaders))
	}

	if len(config.RecvBody) > 0 {
		checks = append(checks, checkBody(config.RecvBody))
	}

	return checkAll(checks...)
}

func checkOK(_ *http.Response) error { return nil }

// TODO: collect all errors into on error message.
func checkAll(checks ...RespCheck) RespCheck {
	switch len(checks) {
	case 0:
		return checkOK
	case 1:
		return checks[0]
	}

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

func checkStatusOK(r *http.Response) error {
	if r.StatusCode >= 400 {
		return errors.New(r.Status)
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

func checkBody(body []match.Matcher) RespCheck {
	return func(r *http.Response) error {
		content, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return err
		}
		for _, m := range body {
			if m.Match(content) {
				return nil
			}
		}
		return errBodyMismatch
	}
}
