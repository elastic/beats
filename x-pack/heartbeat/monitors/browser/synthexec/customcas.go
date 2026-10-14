// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux || darwin || synthetics

package synthexec

import (
	"context"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

type customCAsKey string

// CustomCAsKey is the context key under which a job passes the CA certificates
// (inline PEM or file paths, as []string) that the browser must trust for that job.
var CustomCAsKey = customCAsKey("custom_cas")

// applyCustomCAs makes the browser launched by cmd trust the CAs in ctx. Chromium
// on Linux only reads the NSS database in $HOME/.pki/nssdb, so it builds one in a
// throwaway home directory and points the command's HOME at it. The CAs therefore
// apply to this run only. The returned cleanup removes the directory and is always
// safe to call, even when there is nothing to do.
func applyCustomCAs(ctx context.Context, cmd *SynthCmd) (cleanup func(), err error) {
	noop := func() {}
	sources, _ := ctx.Value(CustomCAsKey).([]string)
	if len(sources) == 0 {
		return noop, nil
	}

	certutil, err := exec.LookPath("certutil")
	if err != nil {
		return noop, fmt.Errorf("certutil is required to trust custom certificate authorities in browser monitors, install nss-tools (libnss3-tools): %w", err)
	}

	home, err := os.MkdirTemp("", "heartbeat-ca-")
	if err != nil {
		return noop, fmt.Errorf("could not create trust store directory: %w", err)
	}
	cleanup = func() { _ = os.RemoveAll(home) }
	defer func() {
		if err != nil {
			cleanup()
			cleanup = noop
		}
	}()

	db := filepath.Join(home, ".pki", "nssdb")
	if err = os.MkdirAll(db, 0o700); err != nil {
		return cleanup, fmt.Errorf("could not create NSS database directory: %w", err)
	}
	dbArg := "sql:" + db
	userHome := os.Getenv("HOME")
	// Start from the shared database, if any, so CAs trusted for every browser monitor
	// (e.g. imported by the Elastic Agent image) stay trusted next to this monitor's.
	seeded, err := seedFromSharedDB(filepath.Join(userHome, ".pki", "nssdb"), db)
	if err != nil {
		return cleanup, fmt.Errorf("could not copy the shared NSS database: %w", err)
	}
	if !seeded {
		if err = runCertutil(ctx, certutil, "-N", "-d", dbArg, "--empty-password"); err != nil {
			return cleanup, err
		}
	}

	n := 0
	for _, src := range sources {
		data, rerr := tlscommon.ReadPEMFile(logp.L(), src, "")
		if rerr != nil {
			return cleanup, fmt.Errorf("could not read certificate authority %q: %w", src, rerr)
		}
		// certutil imports only the first certificate of a file, so import them one by one.
		for rest := data; len(rest) > 0; {
			var block *pem.Block
			block, rest = pem.Decode(rest)
			if block == nil {
				break
			}
			if block.Type != "CERTIFICATE" {
				continue
			}
			n++
			file := filepath.Join(home, fmt.Sprintf("ca-%d.pem", n))
			if err = os.WriteFile(file, pem.EncodeToMemory(block), 0o600); err != nil {
				return cleanup, fmt.Errorf("could not write certificate authority: %w", err)
			}
			if err = runCertutil(ctx, certutil, "-A", "-d", dbArg, "-n", fmt.Sprintf("monitor-ca-%d", n), "-t", "C,,", "-i", file); err != nil {
				return cleanup, fmt.Errorf("could not import certificate authority from %q: %w", src, err)
			}
		}
	}
	if n == 0 {
		err = fmt.Errorf("no certificates found in ssl.certificate_authorities")
		return cleanup, err
	}

	// The node process may run as another user than heartbeat.
	if cmd.SysProcAttr != nil && cmd.SysProcAttr.Credential != nil {
		cred := cmd.SysProcAttr.Credential
		if err = chownTree(home, int(cred.Uid), int(cred.Gid)); err != nil {
			return cleanup, fmt.Errorf("could not make trust store readable by the browser user: %w", err)
		}
	}

	// A new HOME must not hide what the browser and its fonts load from the real
	// one: Playwright's browsers and the font cache, which is slow to rebuild.
	cmd.Env = append(cmd.Env, "HOME="+home)
	if userHome != "" {
		if os.Getenv("PLAYWRIGHT_BROWSERS_PATH") == "" {
			cmd.Env = append(cmd.Env, "PLAYWRIGHT_BROWSERS_PATH="+filepath.Join(userHome, ".cache", "ms-playwright"))
		}
		if os.Getenv("XDG_CACHE_HOME") == "" {
			cmd.Env = append(cmd.Env, "XDG_CACHE_HOME="+filepath.Join(userHome, ".cache"))
		}
	}

	return cleanup, nil
}

// seedFromSharedDB copies the NSS database files of src into dst. It reports false,
// without error, when src holds no database.
func seedFromSharedDB(src, dst string) (bool, error) {
	if _, err := os.Stat(filepath.Join(src, "cert9.db")); err != nil {
		if os.IsNotExist(err) || errors.Is(err, os.ErrPermission) {
			return false, nil
		}
		return false, err
	}
	for _, name := range []string{"cert9.db", "key4.db", "pkcs11.txt"} {
		data, err := os.ReadFile(filepath.Join(src, name))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return false, err
		}
		if err := os.WriteFile(filepath.Join(dst, name), data, 0o600); err != nil {
			return false, err
		}
	}
	return true, nil
}

func runCertutil(ctx context.Context, certutil string, args ...string) error {
	if out, err := exec.CommandContext(ctx, certutil, args...).CombinedOutput(); err != nil { //nolint:gosec // certutil path comes from LookPath, args are built here
		return fmt.Errorf("certutil %v failed: %w: %s", args[:1], err, out)
	}
	return nil
}

func chownTree(root string, uid, gid int) error {
	return filepath.WalkDir(root, func(path string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return os.Chown(path, uid, gid)
	})
}
