package dft

import (
	"path"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
)

const esDFTPath = "/_data_frame/transforms"

type ClientHandler interface {
	CheckDataFramesEnabled(Mode) (bool, error)
	EnsureDataFrameTransforms(transforms []*DataFrameTransform) error
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

func (*FileClientHandler) EnsureDataFrameTransforms(transforms []*DataFrameTransform) error {
	panic("implement me ensure")
}

type ESClientHandler struct {
	client ESClient
}

func (*ESClientHandler) CheckDataFramesEnabled(Mode) (bool, error) {
	//TODO make this actually do the thing
	return true, nil
}

func (h *ESClientHandler) EnsureDataFrameTransforms(transforms []*DataFrameTransform) error {
	for _, t := range transforms {
		err := h.EnsureDataFrameTransform(t)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *ESClientHandler) EnsureDataFrameTransform(t *DataFrameTransform) error {
	code, _, err := h.GetDataFrame(t)
	p := path.Join(esDFTPath, t.Name)
	if code == 200 { // Stop existing transform
		err = h.StopDataFrame(t)
		if err != nil {
			return err
		}

		err = h.DeleteDataFrame(t)
		if err != nil {
			return err
		}
	} else if code != 404 {
		return errors.Wrapf(err, "unexpected error checking dataframe at path %v", p)
	}

	err = h.PutDataFrame(t)
	if err != nil {
		return err
	}
	err = h.StartDataFrame(t)
	return err
}

func (h *ESClientHandler) GetDataFrame(t *DataFrameTransform) (code int, body string, err error) {
	code, bodyRaw, err := h.client.Request("GET", t.path(), "", nil, nil)

	return code, string(bodyRaw), err
}

func (h *ESClientHandler) PutDataFrame(t *DataFrameTransform) error {
	body := map[string]interface{}{}
	body["pivot"] = t.Pivot
	body["source"] = map[string]string{"index": t.Source}
	body["dest"] = map[string]string{"index": t.Dest}

	_, _, err := h.client.Request("PUT", t.path(), "", nil, body)

	if err != nil {
		return errors.Wrapf(err, "could not PUT dataframe at path %v", t.path())
	}

	return nil
}

func (h *ESClientHandler) DeleteDataFrame(t *DataFrameTransform) error {
	_, _, err := h.client.Request("DELETE", t.path(), "", nil, nil)
	return err
}

func (h *ESClientHandler) StartDataFrame(t *DataFrameTransform) error {
	_, _, err := h.client.Request("POST", path.Join(t.path(), "_start"), "", nil, nil)
	return err
}

func (h *ESClientHandler) StopDataFrame(t *DataFrameTransform) error {
	_, _, err := h.client.Request("POST", path.Join(t.path(), "_stop"), "", nil, nil)
	return err
}

// NewESClientHandler initializes and returns an ESClientHandler,
func NewESClientHandler(c ESClient) *ESClientHandler {
	return &ESClientHandler{client: c}
}

// NewFileClientHandler initializes and returns a new FileClientHandler instance.
func NewFileClientHandler(c FileClient) *FileClientHandler {
	return &FileClientHandler{client: c}
}
