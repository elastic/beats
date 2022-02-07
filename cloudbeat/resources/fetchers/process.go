package fetchers

import (
	"context"

	"github.com/elastic/beats/v7/cloudbeat/resources"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/proc"
)

const (
	ProcessType = "process"
)

type ProcessesFetcher struct {
	cfg ProcessFetcherConfig
}

type ProcessFetcherConfig struct {
	resources.BaseFetcherConfig
	Directory string `config:"directory"` // parent directory of target procfs
}

func NewProcessesFetcher(cfg ProcessFetcherConfig) resources.Fetcher {
	return &ProcessesFetcher{
		cfg: cfg,
	}
}

func (f *ProcessesFetcher) Fetch(ctx context.Context) ([]resources.FetcherResult, error) {
	pids, err := proc.List(f.cfg.Directory)
	if err != nil {
		return nil, err
	}

	ret := make([]resources.FetcherResult, 0)

	// If errors occur during read, then return what we have till now
	// without reporting errors.
	for _, p := range pids {
		cmd, err := proc.ReadCmdLine(f.cfg.Directory, p)
		if err != nil {
			return ret, nil
		}

		stat, err := proc.ReadStat(f.cfg.Directory, p)
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
