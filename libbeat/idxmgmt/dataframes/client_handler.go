package dataframes

import (
	"github.com/elastic/beats/libbeat/common"
)

type ClientHandler interface {
	CheckDataFramesEnabled(Mode) (bool, error)
	EnsureDataFrames() (bool, error)
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

func (*FileClientHandler) EnsureDataFrames() (bool, error) {
	panic("implement me ensure")
}

type ESClientHandler struct {
	client ESClient
}

func (*ESClientHandler) CheckDataFramesEnabled(Mode) (bool, error) {
	panic("implement me checkes")
}

func (*ESClientHandler) EnsureDataFrames() (bool, error) {
	panic("implement me ensurees")
}

// NewESClientHandler initializes and returns an ESClientHandler,
func NewESClientHandler(c ESClient) *ESClientHandler {
	return &ESClientHandler{client: c}
}

// NewFileClientHandler initializes and returns a new FileClientHandler instance.
func NewFileClientHandler(c FileClient) *FileClientHandler {
	return &FileClientHandler{client: c}
}
