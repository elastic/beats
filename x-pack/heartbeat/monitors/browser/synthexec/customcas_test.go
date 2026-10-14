// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux || darwin || synthetics

package synthexec

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
)

func testCAPEM(t *testing.T, cn string) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err, "generating key")
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err, "creating certificate")
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

// stubCertutil puts a certutil on PATH that records its arguments, one call per line.
func stubCertutil(t *testing.T) (callsFile string) {
	t.Helper()
	dir := t.TempDir()
	callsFile = filepath.Join(dir, "calls")
	script := "#!/bin/sh\necho \"$@\" >> " + callsFile + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "certutil"), []byte(script), 0o755), "writing stub certutil") //nolint:gosec // test stub must be executable
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return callsFile
}

func envValue(env []string, key string) (string, bool) {
	val := ""
	found := false
	for _, e := range env {
		if v, ok := strings.CutPrefix(e, key+"="); ok {
			val, found = v, true
		}
	}
	return val, found
}

func TestApplyCustomCAs(t *testing.T) {
	t.Run("does nothing without CAs", func(t *testing.T) {
		cmd := &SynthCmd{Cmd: exec.Command("true")}
		cleanup, err := applyCustomCAs(context.Background(), cmd)
		require.NoError(t, err, "no CAs must not fail")
		cleanup()
		assert.Empty(t, cmd.Env, "environment must be untouched without CAs")
	})

	t.Run("imports every certificate into a throwaway HOME", func(t *testing.T) {
		calls := stubCertutil(t)
		t.Setenv("HOME", "/real/home")
		t.Setenv("PLAYWRIGHT_BROWSERS_PATH", "")
		t.Setenv("XDG_CACHE_HOME", "")

		inline := testCAPEM(t, "inline ca")
		bundle := testCAPEM(t, "bundle ca 1") + testCAPEM(t, "bundle ca 2")
		bundleFile := filepath.Join(t.TempDir(), "bundle.pem")
		require.NoError(t, os.WriteFile(bundleFile, []byte(bundle), 0o600), "writing bundle")

		ctx := context.WithValue(context.Background(), CustomCAsKey, []string{inline, bundleFile})
		cmd := &SynthCmd{Cmd: exec.Command("true")}
		cleanup, err := applyCustomCAs(ctx, cmd)
		require.NoError(t, err, "applying CAs")

		home, ok := envValue(cmd.Env, "HOME")
		require.True(t, ok, "HOME must be set for the browser")
		assert.NotEqual(t, "/real/home", home, "HOME must point at the throwaway trust store")
		assert.DirExists(t, filepath.Join(home, ".pki", "nssdb"), "NSS database directory must exist where Chromium looks")

		browsers, _ := envValue(cmd.Env, "PLAYWRIGHT_BROWSERS_PATH")
		assert.Equal(t, "/real/home/.cache/ms-playwright", browsers, "browsers must still be found under the real HOME")
		cache, _ := envValue(cmd.Env, "XDG_CACHE_HOME")
		assert.Equal(t, "/real/home/.cache", cache, "caches must still come from the real HOME")

		out, err := os.ReadFile(calls)
		require.NoError(t, err, "reading certutil calls")
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		require.Len(t, lines, 4, "expected one database creation and one import per certificate: %v", lines)
		assert.Contains(t, lines[0], "-N", "database must be created first")
		for i, l := range lines[1:] {
			assert.Contains(t, l, "-A", "certificate %d must be imported", i+1)
			assert.Contains(t, l, "-t C,,", "certificate %d must be trusted as a CA", i+1)
		}

		cleanup()
		assert.NoDirExists(t, home, "cleanup must remove the throwaway HOME")
	})

	t.Run("fails clearly when certutil is missing", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		ctx := context.WithValue(context.Background(), CustomCAsKey, []string{testCAPEM(t, "ca")})
		cleanup, err := applyCustomCAs(ctx, &SynthCmd{Cmd: exec.Command("true")})
		cleanup()
		require.Error(t, err, "missing certutil must fail the run")
		assert.Contains(t, err.Error(), "nss-tools", "the error must say what to install")
	})

	t.Run("fails when no certificate can be read", func(t *testing.T) {
		stubCertutil(t)
		ctx := context.WithValue(context.Background(), CustomCAsKey, []string{"/does/not/exist.pem"})
		cleanup, err := applyCustomCAs(ctx, &SynthCmd{Cmd: exec.Command("true")})
		cleanup()
		require.Error(t, err, "an unreadable CA must fail the run")
	})
}

// TestApplyCustomCAsRealCertutil checks the trust store with the real certutil,
// since the stubbed test can't tell whether the database is valid.
func TestApplyCustomCAsRealCertutil(t *testing.T) {
	certutil, err := exec.LookPath("certutil")
	if err != nil {
		t.Skip("certutil not installed")
	}

	ctx := context.WithValue(context.Background(), CustomCAsKey, []string{testCAPEM(t, "one") + testCAPEM(t, "two")})
	cmd := &SynthCmd{Cmd: exec.Command("true")}
	cleanup, err := applyCustomCAs(ctx, cmd)
	require.NoError(t, err, "applying CAs")
	defer cleanup()

	home, _ := envValue(cmd.Env, "HOME")
	out, err := exec.Command(certutil, "-L", "-d", "sql:"+filepath.Join(home, ".pki", "nssdb")).CombinedOutput() //nolint:gosec // test
	require.NoError(t, err, "listing the database: %s", out)
	assert.Contains(t, string(out), "monitor-ca-1", "first certificate must be in the database")
	assert.Contains(t, string(out), "monitor-ca-2", "second certificate of the bundle must be in the database")
}

func TestRunCmdKeepsCustomCAsUntilCommandExits(t *testing.T) {
	stubCertutil(t)
	record := filepath.Join(t.TempDir(), "record")
	// Outlive the start of the command to catch a cleanup that runs too early.
	script := `sleep 1; echo "$HOME" > ` + record + `; test -d "$HOME/.pki/nssdb" && echo present >> ` + record
	ctx := context.WithValue(context.Background(), SynthexecTimeoutKey, time.Minute)
	ctx = context.WithValue(ctx, CustomCAsKey, []string{testCAPEM(t, "ca")})

	cmd := exec.Command("sh", "-c", script)
	stdin := ""
	mpx, err := runCmd(ctx, &SynthCmd{cmd}, &stdin, nil, FilterJourneyConfig{}, stdfields.StdMonitorFields{}, "")
	require.NoError(t, err, "running command with custom CAs")
	for range mpx.SynthEvents() { //nolint:revive // drain until the command is done
	}

	out, err := os.ReadFile(record)
	require.NoError(t, err, "reading what the command saw")
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	require.Len(t, lines, 2, "the trust store must still exist while the command runs: %q", out)
	assert.Equal(t, "present", lines[1], "the NSS database must exist for the command's whole run")
	assert.Eventually(t, func() bool { _, err := os.Stat(lines[0]); return os.IsNotExist(err) }, 5*time.Second, 50*time.Millisecond,
		"the trust store must be removed once the command exits")
}

func TestApplyCustomCAsSeedsFromSharedDB(t *testing.T) {
	calls := stubCertutil(t)
	userHome := t.TempDir()
	shared := filepath.Join(userHome, ".pki", "nssdb")
	require.NoError(t, os.MkdirAll(shared, 0o700), "creating shared db dir")
	require.NoError(t, os.WriteFile(filepath.Join(shared, "cert9.db"), []byte("shared certs"), 0o600), "writing shared cert db")
	require.NoError(t, os.WriteFile(filepath.Join(shared, "key4.db"), []byte("shared keys"), 0o600), "writing shared key db")
	t.Setenv("HOME", userHome)

	ctx := context.WithValue(context.Background(), CustomCAsKey, []string{testCAPEM(t, "ca")})
	cmd := &SynthCmd{Cmd: exec.Command("true")}
	cleanup, err := applyCustomCAs(ctx, cmd)
	require.NoError(t, err, "applying CAs")
	defer cleanup()

	home, _ := envValue(cmd.Env, "HOME")
	got, err := os.ReadFile(filepath.Join(home, ".pki", "nssdb", "cert9.db"))
	require.NoError(t, err, "reading the seeded cert db")
	assert.Equal(t, "shared certs", string(got), "the shared database must be the starting point")

	out, err := os.ReadFile(calls)
	require.NoError(t, err, "reading certutil calls")
	assert.NotContains(t, string(out), "-N", "an existing shared database must not be recreated empty")
	assert.Contains(t, string(out), "-A", "the monitor's CA must still be added")

	again, err := os.ReadFile(filepath.Join(shared, "cert9.db"))
	require.NoError(t, err, "reading the shared db again")
	assert.Equal(t, "shared certs", string(again), "the shared database itself must not be modified")
}

// TestApplyCustomCAsKeepsSharedCAs checks with the real certutil that a CA trusted
// through the shared database and one set on the monitor are trusted together.
func TestApplyCustomCAsKeepsSharedCAs(t *testing.T) {
	certutil, err := exec.LookPath("certutil")
	if err != nil {
		t.Skip("certutil not installed")
	}

	userHome := t.TempDir()
	shared := filepath.Join(userHome, ".pki", "nssdb")
	require.NoError(t, os.MkdirAll(shared, 0o700), "creating shared db dir")
	sharedArg := "sql:" + shared
	out, err := exec.Command(certutil, "-N", "-d", sharedArg, "--empty-password").CombinedOutput() //nolint:gosec // test
	require.NoError(t, err, "creating shared db: %s", out)
	caFile := filepath.Join(t.TempDir(), "shared.pem")
	require.NoError(t, os.WriteFile(caFile, []byte(testCAPEM(t, "shared")), 0o600), "writing shared CA")
	out, err = exec.Command(certutil, "-A", "-d", sharedArg, "-n", "custom-ca-shared", "-t", "C,,", "-i", caFile).CombinedOutput() //nolint:gosec // test
	require.NoError(t, err, "importing shared CA: %s", out)
	t.Setenv("HOME", userHome)

	ctx := context.WithValue(context.Background(), CustomCAsKey, []string{testCAPEM(t, "monitor")})
	cmd := &SynthCmd{Cmd: exec.Command("true")}
	cleanup, err := applyCustomCAs(ctx, cmd)
	require.NoError(t, err, "applying CAs")
	defer cleanup()

	home, _ := envValue(cmd.Env, "HOME")
	out, err = exec.Command(certutil, "-L", "-d", "sql:"+filepath.Join(home, ".pki", "nssdb")).CombinedOutput() //nolint:gosec // test
	require.NoError(t, err, "listing the monitor database: %s", out)
	assert.Contains(t, string(out), "custom-ca-shared", "the shared CA must stay trusted for the monitor")
	assert.Contains(t, string(out), "monitor-ca-1", "the monitor's own CA must be trusted")

	out, err = exec.Command(certutil, "-L", "-d", sharedArg).CombinedOutput() //nolint:gosec // test
	require.NoError(t, err, "listing the shared database: %s", out)
	assert.NotContains(t, string(out), "monitor-ca-1", "the monitor's CA must not leak into the shared database")
}
