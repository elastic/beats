package useragent

import (
	"fmt"
	"runtime"

	"github.com/elastic/beats/libbeat/version"
)

// UserAgent takes the capitalized name of the current beat and returns
// an RFC compliant user agent string for that beat.
func UserAgent(beatNameCapitalized string) string {
	return fmt.Sprintf("Elastic %s/%s (%s; %s; %s; %s)",
		beatNameCapitalized,
		version.GetDefaultVersion(), runtime.GOOS, runtime.GOARCH,
		version.Commit(), version.BuildTime())
}
