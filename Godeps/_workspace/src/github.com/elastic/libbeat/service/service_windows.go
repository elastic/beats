package service

import (
	"time"

	"github.com/elastic/libbeat/logp"
	"golang.org/x/sys/windows/svc"
)

// On windows this creates a loop that only finishes when
// a Stop or Shutdown request is received. On non-windows
// platforms, the function does nothing. The stopCallback
// function is called when the Stop/Shutdown request is
// received.
func ProcessWindowsControlEvents(stopCallback func()) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for c := range r {
		switch c.Cmd {
		case svg.Interrogate:
			changes <- c.CurrentStatus
			// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
			time.Sleep(100 * time.Millisecond)
			changes <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			stopCallback()
			break loop
		default:
			logp.Err("Unexpected control request: $%d", c)
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}
