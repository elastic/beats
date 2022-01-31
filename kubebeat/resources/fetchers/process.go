package fetchers

import (
	"github.com/elastic/beats/v7/kubebeat/resources"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/proc"
)

const (
	ProcessType = "process"
)

type ProcessesFetcher struct {
	directory string // parent directory of target procfs
}

func NewProcessesFetcher(dir string) resources.Fetcher {
	return &ProcessesFetcher{
		directory: dir,
	}
}

func (f *ProcessesFetcher) Fetch() ([]resources.FetcherResult, error) {
	pids, err := proc.List(f.directory)
	if err != nil {
		return nil, err
	}

	ret := make([]resources.FetcherResult, 0)

	// If errors occur during read, then return what we have till now
	// without reporting errors.
	for _, p := range pids {
		cmd, err := proc.ReadCmdLine(f.directory, p)
		if err != nil {
			return ret, nil
		}

		stat, err := proc.ReadStat(f.directory, p)
		if err != nil {
			return ret, nil
		}

		ret = append(ret, resources.FetcherResult{
			Type:     ProcessType,
			Resource: resources.ProcessResource{PID: p, Cmd: cmd, Stat: stat},
		})
	}

	return ret, nil
}

func (f *ProcessesFetcher) Stop() {
}
