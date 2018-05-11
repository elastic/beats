package ratelimit

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher/scheduling"
)

type policy struct {
	eps    uint
	ticker *time.Ticker
	shared bool
}

func init() {
	scheduling.PolicyRegistry.Register("ratelimit", create)
}

func create(cfg *common.Config) (scheduling.Policy, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	var ticker *time.Ticker
	if config.Shared {
		interval := 1 * time.Second / time.Duration(config.EventsPerSecond)
		ticker = time.NewTicker(interval)
	}

	return &policy{
		eps:    config.EventsPerSecond,
		ticker: ticker,
		shared: config.Shared,
	}, nil
}

func (p *policy) Connect(ctx scheduling.Context) (scheduling.Handler, error) {
	if !p.shared {
		return newLocalHandler(ctx, p.eps), nil
	}
	return newSharedHandler(ctx, p.ticker), nil
}
