package embed

import (
	"github.com/markbates/pkger/here"
	"github.com/markbates/pkger/pkging"
)

type File struct {
	Info   *pkging.FileInfo `json:"info"`
	Here   here.Info        `json:"her"`
	Path   here.Path        `json:"path"`
	Data   []byte           `json:"data"`
	Parent here.Path        `json:"parent"`
}
