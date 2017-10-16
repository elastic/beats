package logstash

import (
	"sync"
	"testing"

	"github.com/elastic/beats/libbeat/logp"
)

var enableLoggingOnce sync.Once

func enableLogging(selectors []string) {
	if testing.Verbose() {
		enableLoggingOnce.Do(func() {
			logp.LogInit(logp.LOG_DEBUG, "", false, true, selectors)
		})
	}
}
