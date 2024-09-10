package panw

import (
	"fmt"
	"strings"
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
