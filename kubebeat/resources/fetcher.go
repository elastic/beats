package resources

import "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/proc"

// Fetcher represents a data fetcher.
type Fetcher interface {
	Fetch() ([]FetcherResult, error)
	Stop()
}

type FetcherResult struct {
	Type     string      `json:"type"`
	Resource interface{} `json:"resource"`
}

type FileSystemResource struct {
	FileName string `json:"filename"`
	FileMode string `json:"mode"`
	Gid      string `json:"gid"`
	Uid      string `json:"uid"`
	Path     string `json:"path"`
}

type ProcessResource struct {
	PID  string        `json:"pid"`
	Cmd  string        `json:"command"`
	Stat proc.ProcStat `json:"stat"`
}
