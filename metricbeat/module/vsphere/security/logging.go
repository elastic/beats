package security

import "github.com/elastic/elastic-agent-libs/logp"

func WarnIfInsecure(logger *logp.Logger, metricSet string, isInsecure bool) {
	if isInsecure {
		logger.With("metricset", metricSet).Warn("Your vSphere connection is configured as insecure. This can lead to man in the middle attack.")
	}
}
