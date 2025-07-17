package ech

import (
	"bytes"
	"context"
	"debug/buildinfo"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/go-elasticsearch/v8"
)

// VerifyEnvVars ensures that the env vars to connect to ES are set, and that the ES_HOST starts with https
func VerifyEnvVars(t *testing.T) {
	t.Helper()
	esHost := os.Getenv("ES_HOST")
	require.NotEmpty(t, esHost, "Expected env var ES_HOST to be not-empty.")
	require.Regexp(t, regexp.MustCompile(`^https://`), esHost)
	esUser := os.Getenv("ES_USER")
	require.NotEmpty(t, esUser, "Expected env var ES_USER to be not-empty.")
	esPass := os.Getenv("ES_PASS")
	require.NotEmpty(t, esPass, "Expected env var ES_PASS to be not-empty.")
}

// VerifyFIPSBinary ensures the binary on binaryPath has FIPS indicators.
func VerifyFIPSBinary(t *testing.T, binaryPath string) {
	t.Helper()
	info, err := buildinfo.ReadFile(binaryPath)
	require.NoError(t, err)

	var checkLinks, foundTags, foundExperiment bool
	for _, setting := range info.Settings {
		switch setting.Key {
		case "-tags":
			foundTags = true
			require.Contains(t, setting.Value, "requirefips")
			require.Contains(t, setting.Value, "ms_tls13kdf")
			continue
		case "GOEXPERIMENT":
			foundExperiment = true
			require.Contains(t, setting.Value, "systemcrypto")
			continue
		case "-ldflags":
			if !strings.Contains(setting.Value, "-s") {
				checkLinks = true
			}
		}
	}

	require.True(t, foundTags, "did not find build tags")
	require.True(t, foundExperiment, "did not find GOEXPERIMENT")

	if checkLinks && runtime.GOOS == "linux" {
		t.Log("Binary is not stripped, checking for OpenSSL in the symbols table.")
		output, err := exec.CommandContext(t.Context(), "go", "tool", "nm", binaryPath).Output()
		require.NoError(t, err, "unable to run go tool nm")
		require.Contains(t, output, "OpenSSL_version", "Unable to find OpenSSL_version in symbols link")
	}
}

// RunSmokeTest runs the beat on binaryPath with the passed config, and ensures that data ends up in Elasticsearch.
func RunSmokeTest(t *testing.T, name, binaryPath, cfg string) {
	// Write config file
	tempDir := t.TempDir()
	configFilePath := path.Join(tempDir, name+".yml")

	err := os.WriteFile(configFilePath, []byte(cfg), 0o644)
	require.NoError(t, err, "unable to write ", configFilePath)

	// start binary
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	ctx, cancel := context.WithCancel(t.Context())
	cmd := exec.CommandContext(ctx, binaryPath, "-c", configFilePath)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	defer func() {
		cancel()
		err := cmd.Wait()
		if t.Failed() {
			t.Logf("%s exited. err: %v\nstdout: %s\nstderr: %s\n", name, err, stdout.String(), stderr.String())
		}
	}()

	err = cmd.Start()
	require.NoError(t, err, "unable to start ", name)

	// ensure data ends up in ES
	es, err := elasticsearch.NewTypedClient(elasticsearch.Config{
		Addresses: []string{os.Getenv("ES_HOST")},
		Username:  os.Getenv("ES_USER"),
		Password:  os.Getenv("ES_PASS"),
	})
	require.NoError(t, err, "unable to create elasticsearch client")

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		resp, err := es.Search().Index(name + "-*").Do(t.Context())
		require.NoError(c, err, "search request for index failed.")
		require.NotZero(c, resp.Hits.Total.Value, "expected to find hits within ES.")
	}, time.Minute, time.Second, name+" logs are not detected within the elasticsearch deployment")
}
