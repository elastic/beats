package main

import (
	"context"
	"os"
	"os/signal"
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
