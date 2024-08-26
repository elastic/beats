package snapshot

import (
	"bytes"
	"io"
	gohttp "net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_checkResponse(t *testing.T) {
	type args struct {
		resp *gohttp.Response
	}
	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Valid http response",
			args: args{
				resp: &gohttp.Response{
					Status:     "OK",
					StatusCode: gohttp.StatusOK,
					Header: map[string][]string{
						"Content-Type": {"application/json; charset=UTF-8"},
					},
					Body: io.NopCloser(strings.NewReader(`{"good": "job"}`)),
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "Bad http status code - 500",
			args: args{
				resp: &gohttp.Response{
					Status:     "Not OK",
					StatusCode: gohttp.StatusInternalServerError,
					Header: map[string][]string{
						"Content-Type": {"application/json"},
					},
					Body: io.NopCloser(strings.NewReader(`{"not feeling": "too well"}`)),
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				retval := assert.ErrorContains(t, err, strconv.Itoa(gohttp.StatusInternalServerError), "error should contain http status code")
				retval = assert.ErrorContains(t, err, `{"not feeling": "too well"}`, "error should contain response body") && retval
				return retval
			},
		},
		{
			name: "Bad http status code - 502",
			args: args{
				resp: &gohttp.Response{
					Status:     "Bad Gateway",
					StatusCode: gohttp.StatusBadGateway,
					Header: map[string][]string{
						"Content-Type": {"application/json; charset=UTF-8"},
					},
					Body: io.NopCloser(strings.NewReader(`{"bad": "gateway"}`)),
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				retval := assert.ErrorContains(t, err, strconv.Itoa(gohttp.StatusBadGateway), "error should contain http status code")
				retval = assert.ErrorContains(t, err, `{"bad": "gateway"}`, "error should contain response body") && retval
				return retval
			},
		},
		{
			name: "Bad http status code - 503",
			args: args{
				resp: &gohttp.Response{
					Status:     "Service Unavailable",
					StatusCode: gohttp.StatusServiceUnavailable,
					Header: map[string][]string{
						"Content-Type": {"application/json"},
					},
					Body: io.NopCloser(bytes.NewReader([]byte{})),
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, strconv.Itoa(gohttp.StatusServiceUnavailable), "error should contain http status code")
			},
		},
		{
			name: "Bad http status code - 504",
			args: args{
				resp: &gohttp.Response{
					Status:     "Gateway timed out",
					StatusCode: gohttp.StatusGatewayTimeout,
					Header: map[string][]string{
						"Content-Type": {"application/json; charset=UTF-8"},
					},
					Body: io.NopCloser(strings.NewReader(`{"gateway": "never got back to me"}`)),
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				retval := assert.ErrorContains(t, err, strconv.Itoa(gohttp.StatusGatewayTimeout), "error should contain http status code")
				retval = assert.ErrorContains(t, err, `{"gateway": "never got back to me"}`, "error should contain response body") && retval
				return retval
			},
		},
		{
			name: "Wrong content type: XML",
			args: args{
				resp: &gohttp.Response{
					Status:     "XML is back in, baby",
					StatusCode: gohttp.StatusOK,
					Header: map[string][]string{
						"Content-Type": {"application/xml"},
					},
					Body: io.NopCloser(strings.NewReader(`<?xml version='1.0' encoding='UTF-8'?><note>Those who cannot remember the past are condemned to repeat it.</note>`)),
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				retval := assert.ErrorContains(t, err, "application/xml")
				retval = assert.ErrorContains(t, err, `<?xml version='1.0' encoding='UTF-8'?><note>Those who cannot remember the past are condemned to repeat it.</note>`)
				return retval
			},
		},
		{
			name: "Wrong content type: HTML",
			args: args{
				resp: &gohttp.Response{
					Status:     "HTML is always (not) machine-friendly",
					StatusCode: gohttp.StatusOK,
					Header: map[string][]string{
						"Content-Type": {"text/html"},
					},
					Body: io.NopCloser(strings.NewReader(`<!DOCTYPE html><html><body>Hello world!</body></html>`)),
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				retval := assert.ErrorContains(t, err, "text/html")
				retval = assert.ErrorContains(t, err, `<!DOCTYPE html><html><body>Hello world!</body></html>`)
				return retval
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkResponse(tt.args.resp)
			if !tt.wantErr(t, err) {
				return
			}
		})
	}
}
