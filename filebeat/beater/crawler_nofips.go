//go:build !requirefips

package beater

import "github.com/elastic/beats/v7/libbeat/cfgfile"

func checkFIPSCapability(_ cfgfile.Runner) error {
	// In non-FIPS builds, we assume all inputs are FIPS capable
	// and proceed without error
	return nil
}
