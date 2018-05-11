package ratelimit

import (
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/publisher/scheduling"
)

type localHandler struct {
	ctx    scheduling.Context
	ticker *time.Ticker
}

func newLocalHandler(ctx scheduling.Context, eps uint) *localHandler {
	interval := 1 * time.Second / time.Duration(eps)
	ticker := time.NewTicker(interval)
	return &localHandler{
		ticker: ticker,
		ctx:    ctx,
	}
}

func (h *localHandler) Close() {
	h.ticker.Stop()
}

func (h *localHandler) OnEvent(evt beat.Event) (beat.Event, error) {
	select {
	case <-h.ctx.Done(): // unblock on close and return closing signal
		return evt, scheduling.SigClose
	case <-h.ticker.C: // take token
		return evt, nil
	}
}
