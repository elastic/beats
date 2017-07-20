package cfgwarn

import (
	"fmt"

	"github.com/elastic/beats/libbeat/logp"
)

// Beta logs the usage of an beta feature.
func Beta(format string, v ...interface{}) {
	logp.Warn("BETA: "+format, v...)
}

// Deprecate logs a deprecation message.
// The version string contains the version when the future will be removed
func Deprecate(version string, format string, v ...interface{}) {
	postfix := fmt.Sprintf(" Will be removed in version: %s", version)
	logp.Warn("DEPRECATED: "+format+postfix, v...)
}

// Experimental logs the usage of an experimental feature.
func Experimental(format string, v ...interface{}) {
	logp.Warn("EXPERIMENTAL: "+format, v...)
}
