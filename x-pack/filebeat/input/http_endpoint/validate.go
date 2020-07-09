package http_endpoint

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

type validator interface {
	// ValidateHeader checks the HTTP headers for compliance. The body must not
	// be touched.
	ValidateHeader(*http.Request) (int, error)
}

type apiValidator struct {
	basicAuth          bool
	username, password string
	method             string
	contentType        string
}

var errIncorrectUserOrPass = errors.New("Incorrect username or password")

func (v *apiValidator) ValidateHeader(r *http.Request) (int, error) {
	if v.basicAuth {
		username, password, _ := r.BasicAuth()
		if v.username != username || v.password != password {
			return http.StatusUnauthorized, errIncorrectUserOrPass
		}
	}

	if v.method != "" && v.method != r.Method {
		return http.StatusMethodNotAllowed, fmt.Errorf("Only %v requests are supported", v.method)
	}

	if v.contentType != "" && r.Header.Get("Content-Type") != v.contentType {
		return http.StatusUnsupportedMediaType, fmt.Errorf("Wrong Content-Type header, expecting %v", v.contentType)
	}

	return 0, nil
}

func withValidator(v validator, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if status, err := v.ValidateHeader(r); status != 0 && err != nil {
			w.WriteHeader(status)
			io.WriteString(w, err.Error())
			return
		}
		handler(w, r)
	}
}
