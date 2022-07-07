package esutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/elastic/go-elasticsearch/v8/esapi"
)

func ToJsonRdr(i interface{}) (io.Reader, error) {
	b, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(b), nil
}

func CheckResp(r *esapi.Response, argErr error) error {
	if argErr != nil {
		return argErr
	}
	if r.StatusCode >= 300 {
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(r.Body)
		if err != nil {
			buf.WriteString(fmt.Sprintf("<error reading body string: %s>", err))
		}
		return fmt.Errorf("bad status code for response(%d): %s", r.StatusCode, buf.String())
	}
	return nil
}

func CheckRetResp(r *esapi.Response, argErr error) (body []byte, err error) {
	if argErr != nil {
		return nil, argErr
	}
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(r.Body)
	if err != nil {
		return nil, fmt.Errorf("<error reading body string: %s>", err)
	}

	if r.StatusCode >= 300 {
		return nil, fmt.Errorf("bad status code for response(%d): %s", r.StatusCode, buf.String())
	}

	return buf.Bytes(), err
}
