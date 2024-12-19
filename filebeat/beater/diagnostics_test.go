package beater

import (
	"fmt"
	"testing"
)

func TestMatchRegistryFiles(t *testing.T) {
	positiveMatches := []string{
		"registry/filebeat/49855.json",
		"registry/filebeat/active.dat",
		"registry/filebeat/meta.json",
		"registry/filebeat/log.json",
	}
	negativeMatches := []string{
		"registry/filebeat/bar.dat",
		"registry/filebeat/log.txt",
		"registry/42.json",
		"nop/active.dat",
	}

	testFn := func(t *testing.T, path string, match bool) {
		result := matchRegistyFiles(path)
		if result != match {
			t.Errorf(
				"mathRegisryFiles('%s') should return %t, got %t instead",
				path,
				match,
				result)
		}
	}

	for _, path := range positiveMatches {
		t.Run(fmt.Sprintf("%s returns true", path), func(t *testing.T) {
			testFn(t, path, true)
		})
	}

	for _, path := range negativeMatches {
		t.Run(fmt.Sprintf("%s returns false", path), func(t *testing.T) {
			testFn(t, path, false)
		})
	}
}
