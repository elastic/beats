package logstash

import (
	"testing"

	"github.com/elastic/beats/libbeat/logp"
)

func enableLogging(selectors []string) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, selectors)
	}
}
