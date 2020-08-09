package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/timed"
	"github.com/elastic/go-concert/unison"
)

// osSignalContext creates a context.Context that will be cancelled if the
// configured os signals are received. osSignalContext exits the process
// immediately with error code 3 if the signal is received a second time.
// Calling the cancel function triggers the context cancellation and stops
// the os signal listener.
func osSignalContext(sigs ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	go func() {
		defer func() {
			signal.Stop(ch)
			cancel()
		}()

		select {
		case <-ctx.Done():
			return
		case <-ch:
			cancel()
			// force shutdown in case we receive another signal
			<-ch
			os.Exit(3)
		}
	}()

	signal.Notify(ch, sigs...)
	return ctx, cancel
}

//periodic wraps timed.Period to provide an error return and cancel the loop
// if fn returns an error.
//
// XXX: elastic/go-concert#28 updated timed.Period to match the interface of
// periodic. This function should be removed when updating to a newer version
// of go-concert.
func periodic(cancel unison.Canceler, period time.Duration, fn func() error) error {
	ctx, cancelFn := context.WithCancel(ctxtool.FromCanceller(cancel))
	defer cancelFn()

	var err error
	timed.Periodic(ctx, period, func() {
		err = fn()
		if err != nil {
			cancelFn()
		}
	})

	if err == nil {
		err = ctx.Err()
	}
	return err
}
