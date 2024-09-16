package panw

import (
	"fmt"
	"strings"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func StringToBool(s string) (bool, error) {
	s = strings.ToLower(s)
	switch s {
	case "yes":
		return true, nil
	case "true":
		return true, nil
	case "no":
		return false, nil
	case "false":
		return false, nil
	}

	// Default to false
	return false, fmt.Errorf("invalid value: %s", s)
}

func MakeRootFields(HostIp string) mapstr.M {
	return mapstr.M{
		"observer.ip":     HostIp,
		"host.ip":         HostIp,
		"observer.vendor": "Palo Alto",
		"observer.type":   "firewall",
	}
}
