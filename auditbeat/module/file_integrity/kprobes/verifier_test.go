package kprobes

import (
	"context"
	"github.com/elastic/beats/v7/auditbeat/module/file_integrity/kprobes/tracing"
	"github.com/stretchr/testify/require"
	"os"
	"runtime"
	"testing"
	"time"
)

func Test_getVerifiedProbes(t *testing.T) {

	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skip("skipping on non-amd64/arm64")
	}

	if runtime.GOOS != "linux" {
		t.Skip("skipping on non-linux")
	}

	if os.Getuid() != 0 {
		t.Skip("skipping as non-root")
	}

	tfs, err := tracing.NewTraceFS()
	require.NoError(t, err)

	err = tfs.RemoveAllKProbes()
	require.NoError(t, err)

	probes, _, err := getVerifiedProbes(context.Background(), 5*time.Second)
	require.NoError(t, err)
	require.NotEmpty(t, probes)
}
