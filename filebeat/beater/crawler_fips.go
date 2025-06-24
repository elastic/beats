//go:build requirefips

package beater

import (
	"fmt"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
)

func checkFIPSCapability(runner cfgfile.Runner) error {
	fipsAwareInput, ok := runner.(v2.FIPSAwareInput)
	if !ok {
		// Input is not FIPS-aware; assume it's FIPS capable and proceed
		// without error
		return nil
	}

	if fipsAwareInput.IsFIPSCapable() {
		// Input is FIPS-capable, proceed without error
		return nil
	}

	return fmt.Errorf("running a FIPS-capable distribution but input %s is not FIPS capable", runner.String())
}
