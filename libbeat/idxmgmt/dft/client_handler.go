package dft

import (
	"encoding/json"
	"path"

	"fmt"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
)

const esDFTPath = "/_data_frame/transforms"

type ClientHandler interface {
	CheckDataFramesEnabled(Mode) (bool, error)
	EnsureDataFrames(transforms []*DataFrameTransform) error
}

// ESClient defines the minimal interface required for the Loader to
// prepare a policy and write alias.
type ESClient interface {
	GetVersion() common.Version
	Request(
		method, path string,
		pipeline string,
		params map[string]string,
		body interface{},
	) (int, []byte, error)
}

type FileClient interface {
	GetVersion() common.Version
	Write(component string, name string, body string) error
}

// FileClientHandler implements the Loader interface for writing to a file.
type FileClientHandler struct {
	client FileClient
}

func (*FileClientHandler) CheckDataFramesEnabled(Mode) (bool, error) {
	panic("implement me check")
}

func (*FileClientHandler) EnsureDataFrames(transforms []*DataFrameTransform) error {
	panic("implement me ensure")
}

type ESClientHandler struct {
	client ESClient
}

func (*ESClientHandler) CheckDataFramesEnabled(Mode) (bool, error) {
	//TODO make this actually do the thing
	return true, nil
}

func (h *ESClientHandler) EnsureDataFrames(transforms []*DataFrameTransform) error {
	for _, t := range transforms {
		p := path.Join(esDFTPath, t.Name)
		code, _, err := h.client.Request("GET", p, "", nil, nil)
		if code == 200 { // Stop existing transform
			code, _, err = h.client.Request("POST", path.Join(p, "_stop"), "", nil, nil)
			if err != nil {
				return err
			}

			_, _, err := h.client.Request("DELETE", p, "", nil, nil)
			if err != nil {
				return err
			}
		} else if code != 404 {
			return errors.Wrapf(err, "unexpected error checking dataframe at path %v", p)
		}

		body := map[string]interface{}{}
		body["pivot"] = t.Pivot
		body["source"] = map[string]string{"index": t.Source}
		body["dest"] = map[string]string{"index": t.Dest}
		j, _ := json.Marshal(body)
		fmt.Printf("DFT BODY %s\n", string(j))

		code, _, err = h.client.Request("PUT", p, "", nil, body)
		if err != nil {
			return errors.Wrapf(err, "could not PUT dataframe at path %v", p)
		}
		fmt.Printf("Created %v with code %v", p, code)

		_, _, err = h.client.Request("POST", path.Join(p, "_start"), "", nil, nil)

		return err
	}
	return nil
}

// NewESClientHandler initializes and returns an ESClientHandler,
func NewESClientHandler(c ESClient) *ESClientHandler {
	return &ESClientHandler{client: c}
}

// NewFileClientHandler initializes and returns a new FileClientHandler instance.
func NewFileClientHandler(c FileClient) *FileClientHandler {
	return &FileClientHandler{client: c}
}
