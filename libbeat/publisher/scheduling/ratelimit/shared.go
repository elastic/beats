package ratelimit

import (
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/publisher/scheduling"
)

type sharedHandler struct {
	ctx    scheduling.Context
	ticker *time.Ticker
}

func newSharedHandler(ctx scheduling.Context, ticker *time.Ticker) *sharedHandler {
	return &sharedHandler{ctx: ctx, ticker: ticker}
}

func (h *sharedHandler) OnEvent(evt beat.Event) (beat.Event, error) {
	select {
	case <-h.ctx.Done(): // unblock on local close and return closing signal
		return evt, scheduling.SigClose
	case <-h.ticker.C: // take token
		return evt, nil
	}
}
