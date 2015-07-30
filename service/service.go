package service

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/elastic/libbeat/logp"
)

// Handles OS signals that ask the service/daemon to stop.
// The stopFunction should break the loop in the Beat so that
// the service shut downs gracefully.
func HandleSignals(stopFunction func()) {
	// On ^C or SIGTERM, gracefully stop the sniffer
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigc
		logp.Debug("service", "Received sigterm/sigint, stopping")
		stopFunction()
	}()

	// Handle the Windows service events
	go ProcessWindowsControlEvents(func() {
		logp.Debug("service", "Received svc stop/shutdown request")
		stopFunction()
	})
}
