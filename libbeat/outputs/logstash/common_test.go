package logstash

import (
	"github.com/elastic/beats/libbeat/logp"
)

func enableLogging(selectors []string) {
	logp.TestingSetup(logp.WithSelectors(selectors...))
}
