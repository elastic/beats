package client

import (
	"encoding/json"
	"fmt"
	"io"
)

type ErrorResponse struct {
	Info *ErrorInfo `json:"error,omitempty"`
}

type ErrorInfo struct {
	RootCause []*ErrorInfo
	Type      string
	Reason    string
	Phase     string
}

func errorFromBody(body io.Reader) error {
	var e ErrorResponse
	d := json.NewDecoder(body)
	if err := d.Decode(&e); err != nil {
		return err
	}

	return fmt.Errorf("elasticsearch error: %+#v", e)
}
