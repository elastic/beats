package metricset

import (
	"os"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

func handleFatalErrors(logger *logp.Logger, errChan chan error, errorCode int) {
	for err := range errChan {
		logger.Error(err)
		// sleep is needed to make sure the error is logged and error event is sent before exiting
		time.Sleep(time.Second * 5)
		os.Exit(errorCode)
	}
}
