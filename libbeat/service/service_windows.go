package service

import (
	"os"
	"time"

	"github.com/elastic/libbeat/logp"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
)

type beatService struct{}

func (m *beatService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {

	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for c := range r {
		switch c.Cmd {
		case svc.Interrogate:
			changes <- c.CurrentStatus
			// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
			time.Sleep(100 * time.Millisecond)
			changes <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			break loop
		default:
			logp.Err("Unexpected control request: $%d. Ignored.", c)
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

// On windows this creates a loop that only finishes when
// a Stop or Shutdown request is received. On non-windows
// platforms, the function does nothing. The stopCallback
// function is called when the Stop/Shutdown request is
// received.
func ProcessWindowsControlEvents(stopCallback func()) {
	isInteractive, err := svc.IsAnInteractiveSession()
	if err != nil {
		logp.Err("IsAnInteractiveSession: %v", err)
		return
	}
	logp.Debug("service", "Windows is interactive: %v", isInteractive)

	run := svc.Run
	if isInteractive {
		run = debug.Run
	}
	err = run(os.Args[0], &beatService{})
	if err != nil {
		logp.Err("Error: %v", err)
	} else {
		stopCallback()
	}
}
