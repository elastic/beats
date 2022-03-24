//go:build freebsd || linux
// +build freebsd linux

package filesystem

import (
	"fmt"
	"os"
	"strings"
)

// actually get the list of mounts on linux
func parseMounts(path string, filter func(FSStat) bool) ([]FSStat, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading mount file %s: %w", path, err)
	}
	fsList := []FSStat{}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		fs := FSStat{
			Device:    fields[0],
			Directory: fields[1],
			Type:      fields[2],
			Options:   fields[3],
		}
		if filter(fs) {
			fsList = append(fsList, fs)
		}
	}

	return fsList, nil
}
