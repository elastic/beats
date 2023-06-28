//go:build integration

package integration

import (
	// "fmt"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestMetaFileExists(t *testing.T) {
	mockbeat := startMockBeat(t, "mockbeat start running.")
	_, err := os.Stat(mockbeat.TempDir() + "/data/meta.json")
	require.Equal(t, err, nil)
}

func TestMetaFilePermissions(t *testing.T) {
	mockbeat := startMockBeat(t, "mockbeat start running.")
	stat, _ := os.Stat(mockbeat.TempDir() + "/data/meta.json")
	require.Equal(t, stat.Mode().String(), "-rw-------")
}
