// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package artifact

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/gofrs/flock"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/fetch"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/install"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/install/artifactformat"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/install/bundled"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

const (
	releasesDirName      = "releases"
	releaseMetadataFile  = "install.json"
	stagingDirNamePrefix = "staging-"
	installLockFileName  = ".custom-artifact-install.lock"
	installLockRetry     = 250 * time.Millisecond
	httpClientTimeout    = 30 * time.Minute
)

type Result struct {
	BinDir  string
	Version string
}

type metadata struct {
	ArtifactURL string    `json:"artifact_url"`
	SHA256      string    `json:"sha256"`
	Version     string    `json:"version"`
	InstalledAt time.Time `json:"installed_at"`
}

func Ensure(ctx context.Context, cfg config.InstallConfig, installDir string, log *logp.Logger) (Result, error) {
	if err := cfg.NormalizeAndValidate(); err != nil {
		return Result{}, err
	}
	selected, enabled := cfg.SelectedForPlatform(runtime.GOOS, runtime.GOARCH)
	if !enabled {
		return Result{}, fmt.Errorf("custom osquery artifact is not enabled for platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if installDir == "" {
		return Result{}, errors.New("install directory is required")
	}
	if err := os.MkdirAll(installDir, 0750); err != nil {
		return Result{}, err
	}
	unlock, err := acquireInstallLock(ctx, installDir)
	if err != nil {
		return Result{}, err
	}
	defer unlock()

	releaseDir := filepath.Join(installDir, releasesDirName, selected.SHA256)
	if res, ok := tryReuseInstalled(releaseDir, selected, log); ok {
		return res, nil
	}

	stageDir, err := os.MkdirTemp(installDir, stagingDirNamePrefix)
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(stageDir)

	artifactFile := filepath.Join(stageDir, "artifact")
	httpClient, err := buildHTTPClient(cfg, runtime.GOOS, runtime.GOARCH, log)
	if err != nil {
		return Result{}, err
	}
	hashOut, err := fetch.DownloadWithClient(ctx, httpClient, selected.ArtifactURL, artifactFile)
	if err != nil {
		return Result{}, err
	}
	if !strings.EqualFold(hashOut, selected.SHA256) {
		return Result{}, fmt.Errorf("artifact sha256 mismatch: expected %s, got %s", selected.SHA256, hashOut)
	}

	extractedDir := filepath.Join(stageDir, "extract")
	if err := os.MkdirAll(extractedDir, 0750); err != nil {
		return Result{}, err
	}
	if err := extractArtifact(artifactFile, selected.ArtifactURL, extractedDir); err != nil {
		return Result{}, err
	}

	binDir, err := locateBinDir(extractedDir, runtime.GOOS)
	if err != nil {
		return Result{}, err
	}
	version, err := install.VerifyOsqueryBinary(runtime.GOOS, binDir, log)
	if err != nil {
		return Result{}, err
	}

	if err := os.MkdirAll(filepath.Dir(releaseDir), 0750); err != nil {
		return Result{}, err
	}
	if _, statErr := os.Stat(releaseDir); statErr == nil {
		if res, ok := tryReuseInstalled(releaseDir, selected, log); ok {
			return res, nil
		}
		_ = os.RemoveAll(releaseDir)
	}
	plan, err := bundled.BuildRuntimePlan(extractedDir, binDir, runtime.GOOS, releaseDir)
	if err != nil {
		return Result{}, err
	}
	if err := bundled.InstallFromTemp(extractedDir, releaseDir, plan, bundled.CopyPath); err != nil {
		return Result{}, err
	}

	meta := metadata{
		ArtifactURL: selected.ArtifactURL,
		SHA256:      strings.ToLower(selected.SHA256),
		Version:     version,
		InstalledAt: time.Now().UTC(),
	}
	if err := writeMetadata(filepath.Join(releaseDir, releaseMetadataFile), meta); err != nil {
		return Result{}, err
	}
	if err := cleanupOldReleases(installDir, releaseDir); err != nil {
		return Result{}, err
	}

	return Result{
		BinDir:  releaseDir,
		Version: version,
	}, nil
}

// RemoveInstalled removes previously managed custom osquery artifact state.
// It is used when custom artifact config is removed and osquerybeat should
// return to bundled-only mode.
func RemoveInstalled(installDir string) error {
	releasesDir := filepath.Join(installDir, releasesDirName)
	if err := os.RemoveAll(releasesDir); err != nil {
		return fmt.Errorf("failed removing releases directory %s: %w", releasesDir, err)
	}
	return nil
}

func tryReuseInstalled(releaseDir string, cfg config.InstallArtifactConfig, log *logp.Logger) (Result, bool) {
	metaPath := filepath.Join(releaseDir, releaseMetadataFile)
	meta, err := readMetadata(metaPath)
	if err != nil {
		log.Debugf("custom osquery metadata not found in %s: %v", releaseDir, err)
		return Result{}, false
	}
	if !strings.EqualFold(meta.SHA256, cfg.SHA256) {
		return Result{}, false
	}

	binDir, err := locateBinDir(releaseDir, runtime.GOOS)
	if err != nil {
		return Result{}, false
	}
	version, err := install.VerifyOsqueryBinary(runtime.GOOS, binDir, log)
	if err != nil {
		return Result{}, false
	}
	return Result{BinDir: binDir, Version: version}, true
}

func readMetadata(fp string) (metadata, error) {
	b, err := os.ReadFile(fp)
	if err != nil {
		return metadata{}, err
	}
	var m metadata
	if err := json.Unmarshal(b, &m); err != nil {
		return metadata{}, err
	}
	return m, nil
}

func writeMetadata(fp string, m metadata) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp := fp + ".tmp"
	if err := os.WriteFile(tmp, b, 0640); err != nil {
		return err
	}
	return os.Rename(tmp, fp)
}

func buildHTTPClient(cfg config.InstallConfig, goos, goarch string, log *logp.Logger) (*http.Client, error) {
	client := &http.Client{
		Timeout: httpClientTimeout,
	}
	sslConfig := cfg.SSLForPlatform(goos, goarch)
	if sslConfig == nil {
		return client, nil
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	sslCfg, err := tlscommon.LoadTLSConfig(sslConfig, log)
	if err != nil {
		return nil, fmt.Errorf("failed loading osquery.elastic_options.install.%s.ssl config: %w", goos, err)
	}
	transport.TLSClientConfig = sslCfg.ToConfig()
	client.Transport = transport
	return client, nil
}

func acquireInstallLock(ctx context.Context, installDir string) (func(), error) {
	lockPath := filepath.Join(installDir, installLockFileName)
	lock := flock.New(lockPath)

	for {
		locked, err := lock.TryLock()
		if err != nil {
			return nil, fmt.Errorf("failed acquiring custom artifact install lock %s: %w", lockPath, err)
		}
		if locked {
			return func() {
				_ = lock.Unlock()
			}, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled while waiting for custom artifact install lock %s: %w", lockPath, ctx.Err())
		case <-time.After(installLockRetry):
		}
	}
}

func cleanupOldReleases(installDir, currentReleaseDir string) error {
	releasesDir := filepath.Join(installDir, releasesDirName)
	entries, err := os.ReadDir(releasesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		entryPath := filepath.Join(releasesDir, entry.Name())
		if filepath.Clean(entryPath) == filepath.Clean(currentReleaseDir) {
			continue
		}
		if err := os.RemoveAll(entryPath); err != nil {
			return fmt.Errorf("failed to remove previous osquery release %s: %w", entryPath, err)
		}
	}
	return nil
}

func extractArtifact(artifactFile, artifactURL, destinationDir string) error {
	format, err := artifactformat.Detect(artifactURL)
	if err != nil {
		return err
	}
	return bundled.ExtractToTemp(format, artifactFile, destinationDir, nil)
}

func locateBinDir(root, goos string) (string, error) {
	var candidates []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		base := filepath.Base(path)
		switch goos {
		case "windows":
			if base != "osqueryd.exe" {
				return nil
			}
		default:
			if base != "osqueryd" {
				return nil
			}
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		cleanRel := filepath.Clean(relPath)
		if goos == "darwin" {
			darwinSuffix := filepath.Clean(filepath.Join("osquery.app", "Contents", "MacOS", "osqueryd"))
			if strings.HasSuffix(cleanRel, darwinSuffix) {
				prefix := strings.TrimSuffix(cleanRel, darwinSuffix)
				prefix = strings.TrimSuffix(prefix, string(os.PathSeparator))
				if prefix == "" || prefix == "." {
					candidates = append(candidates, root)
					return nil
				}
				candidates = append(candidates, filepath.Join(root, prefix))
			}
			return nil
		}

		resolvedBinPath, err := resolveBinPathWithinRoot(root, path)
		if err != nil {
			return nil
		}
		candidates = append(candidates, filepath.Dir(resolvedBinPath))
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("failed to locate osquery binary in extracted artifact: %s", root)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return len(candidates[i]) < len(candidates[j])
	})
	return candidates[0], nil
}

func resolveBinPathWithinRoot(root, binPath string) (string, error) {
	resolved := binPath
	if fileInfo, err := os.Lstat(binPath); err == nil && (fileInfo.Mode()&os.ModeSymlink) != 0 {
		evaluated, err := filepath.EvalSymlinks(binPath)
		if err != nil {
			return "", err
		}
		resolved = evaluated
	}

	rel, err := filepath.Rel(root, resolved)
	if err != nil {
		return "", err
	}
	rel = filepath.Clean(rel)
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("resolved osquery binary path escapes extracted root: %s", resolved)
	}
	return resolved, nil
}

