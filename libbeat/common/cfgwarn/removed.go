package cfgwarn

import (
	"fmt"
	"strings"

	"github.com/joeshaw/multierror"

	"github.com/elastic/beats/libbeat/common"
)

func CheckRemoved5xSettings(cfg *common.Config, settings ...string) error {
	var errs multierror.Errors
	for _, setting := range settings {
		if err := CheckRemoved5xSetting(cfg, setting); err != nil {
			errs = append(errs, err)
		}
	}

	return errs.Err()
}

// CheckRemoved5xSetting prints a warning if the obsolete setting is used.
func CheckRemoved5xSetting(cfg *common.Config, setting string) error {
	segments := strings.Split(setting, ".")

	L := len(segments)
	name := segments[L-1]
	path := segments[:L-1]

	current := cfg
	for _, p := range path {
		current, _ := current.Child(p, -1)
		if current == nil {
			break
		}
	}

	// full path to setting not available -> setting not found
	if current == nil {
		return nil
	}

	if !current.HasField(name) {
		return nil
	}

	return fmt.Errorf("setting '%v' has been removed", current.PathOf(name))
}
