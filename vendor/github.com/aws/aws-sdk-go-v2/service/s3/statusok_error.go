package s3

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	request "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
)

func copyMultipartStatusOKUnmarhsalError(r *request.Request) {
	b, err := ioutil.ReadAll(r.HTTPResponse.Body)
	if err != nil {
		r.Error = awserr.New("SerializationError", "unable to read response body", err)
		return
	}
	body := bytes.NewReader(b)
	r.HTTPResponse.Body = ioutil.NopCloser(body)
	defer body.Seek(0, io.SeekStart)

	unmarshalError(r)
	if err, ok := r.Error.(awserr.Error); ok && err != nil {
		if err.Code() == "SerializationError" &&
			!strings.Contains(err.Error(), io.EOF.Error()) {
			r.Error = nil
			return
		}
		// if empty payload
		if strings.Contains(err.Error(), io.EOF.Error()) {
			r.HTTPResponse.StatusCode = http.StatusInternalServerError
		} else {
			r.HTTPResponse.StatusCode = http.StatusServiceUnavailable
		}
	}
}
