package filters

import (
	"os"
	"strconv"
	"strings"
)

var env = func() map[string]string {
	m := map[string]string{}

	for _, line := range os.Environ() {
		kv := strings.Split(line, "=")

		k, v := kv[0], kv[1]
		kt, vt := strings.TrimSpace(k), strings.TrimSpace(v)

		if len(kt) == 0 || len(vt) == 0 {
			continue
		}

		switch k {
		case "GO111MODULE":
			continue
		}

		switch v {
		case "true", "TRUE", "false", "FALSE", "null", "nil", "NULL":
			continue
		}

		if _, err := strconv.Atoi(k); err == nil {
			continue
		}
		if _, err := strconv.Atoi(v); err == nil {
			continue
		}

		m[k] = v
	}
	return m
}()
