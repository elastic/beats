package dft

import (
	"fmt"
	"path"
	"time"

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
	err := h.EnsurePipeline(t.Pipeline)
	if err != nil {
		return err
	}

	err = h.EnsureSourceMetaIndex(t)
	if err != nil {
		return err
	}

	err = h.EnsureDestIndex(t)
	if err != nil {
		return err
	}

	code, _, err := h.GetDataFrame(t)
	p := path.Join(esDFTPath, t.Name)
	if code == 200 { // Stop existing transform
		err = h.StopDataFrame(t)
		// stopping is async, so a delay helps. TODO use a more robust wait method
		time.Sleep(time.Second)
		// Ignore these errors for now, in the future we should be more precise
		// If there is an error it most likely means the dataframe isn't started

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

func (h *ESClientHandler) EnsureSourceMetaIndex(t *DataFrameTransform) error {
	return h.EnsureIndex(t.SourceMetaIdx, common.MapStr{})
}

func (h *ESClientHandler) EnsureDestIndex(t *DataFrameTransform) error {
	return h.EnsureIndex(t.DestIdx, t.DestMappings)
}

func (h *ESClientHandler) EnsureIndex(name string, mapping common.MapStr) error {
	code, _, err := h.client.Request("GET", "/"+name, "", nil, nil)
	if code == 404 {
		body := common.MapStr{
			"mappings": mapping,
		}
		_, _, err := h.client.Request("PUT", "/"+name, "", nil, body)
		return errors.Wrapf(err, "error creating index %s", name)
	} else if code >= 200 && code <= 299 {
		// TODO check if mapping is for older or newer version, and only update if we have a newer version
		_, _, err := h.client.Request("PUT", "/"+name+"/_mapping", "", nil, mapping)
		return errors.Wrapf(err, "error updating mapping for index %s", name)
	}

	return errors.Wrapf(err, "error while checking index %s existence", name)
}

func (h *ESClientHandler) EnsurePipeline(p Pipeline) error {
	// TODO only overwrite the pipeline if the new pipeline is a newer versio
	//  n
	body := common.MapStr{
		"description": p.Description,
		"processors":  p.Processors,
	}
	_, _, err := h.client.Request("PUT", fmt.Sprintf("/_ingest/pipeline/%s", p.ID), "", nil, body)
	return err
}

func (h *ESClientHandler) GetDataFrame(t *DataFrameTransform) (code int, body string, err error) {
	code, bodyRaw, err := h.client.Request("GET", t.path(), "", nil, nil)

	return code, string(bodyRaw), err
}

func (h *ESClientHandler) PutDataFrame(t *DataFrameTransform) error {
	body := map[string]interface{}{}
	body["pivot"] = t.Pivot
	body["source"] = map[string]string{"index": t.SourceIdx}
	body["dest"] = map[string]string{"index": t.DestIdx}

	_, _, err := h.client.Request("PUT", t.path(), "", nil, body)

	if err != nil {
		return errors.Wrapf(err, "could not PUT dataframe at path %v", t.path())
	}

	return nil
}

func (h *ESClientHandler) DeleteDataFrame(t *DataFrameTransform) error {
	_, _, err := h.client.Request("DELETE", t.path(), "", nil, nil)
	return errors.Wrapf(err, "could not delete dataframe %s", t.Name)
}

func (h *ESClientHandler) StartDataFrame(t *DataFrameTransform) error {
	_, _, err := h.client.Request("POST", path.Join(t.path(), "_start"), "", nil, nil)
	return errors.Wrapf(err, "could not start dataframe %s", t.Name)
}

func (h *ESClientHandler) StopDataFrame(t *DataFrameTransform) error {
	_, _, err := h.client.Request("POST", path.Join(t.path(), "_stop"), "", map[string]string{"force": "true"}, nil)
	return errors.Wrapf(err, "could not stop dataframe %s", t.Name)
}

// NewESClientHandler initializes and returns an ESClientHandler,
func NewESClientHandler(c ESClient) *ESClientHandler {
	return &ESClientHandler{client: c}
}

// NewFileClientHandler initializes and returns a new FileClientHandler instance.
func NewFileClientHandler(c FileClient) *FileClientHandler {
	return &FileClientHandler{client: c}
}
