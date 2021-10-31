package filters

import (
	"bytes"
	"strings"
)

func Secrets() FilterFunc {
	return func(b []byte) ([]byte, error) {
		for k, v := range env {
			for _, s := range secretSuffixes {
				if !strings.HasSuffix(k, s) {
					continue
				}
				b = bytes.ReplaceAll(b, []byte(v), []byte(mask()))
				break
			}
		}
		return b, nil
	}
}

var secretSuffixes = []string{
	"_KEY",
	"_SECRET",
	"_TOKEN",
	"_PASSWORD",
	"_PASS",
}
