package client

import (
	"github.com/elastic/beats/libbeat/common"
)

// API defines the version-agnostic Elasticsearch API subset needed by Beats
type API interface {
	GetLicense() (*License, error)
	GetVersion() (*common.Version, error)
}
