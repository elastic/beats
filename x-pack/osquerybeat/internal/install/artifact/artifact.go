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

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/fetch"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/install"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/msiutil"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/pkgutil"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/tar"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/zip"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

const (
	releasesDirName      = "releases"
	activeReleaseFile    = "active_release"
	releaseMetadataFile  = "install.json"
	stagingDirNamePrefix = "staging-"
)

type Result struct {
	BinPath string
	Version string
}

type metadata struct {
	ArtifactURL string    `json:"artifact_url"`
	SHA256      string    `json:"sha256"`
	Version     string    `json:"version"`
	InstalledAt time.Time `json:"installed_at"`
}

func Ensure(ctx context.Context, cfg config.InstallConfig, installDir string, log *logp.Logger) (Result, error) {
	if !cfg.Enabled() {
		return Result{}, errors.New("custom osquery artifact is not enabled")
	}
	if err := cfg.Validate(); err != nil {
		return Result{}, err
	}
	if installDir == "" {
		return Result{}, errors.New("install directory is required")
	}
	if err := os.MkdirAll(installDir, 0750); err != nil {
		return Result{}, err
	}

	releaseDir := filepath.Join(installDir, releasesDirName, cfg.SHA256)
	if res, ok := tryReuseInstalled(releaseDir, cfg, log); ok {
		_ = writeActiveReleaseFile(installDir, releaseDir)
		return res, nil
	}

	stageDir, err := os.MkdirTemp(installDir, stagingDirNamePrefix)
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(stageDir)

	artifactFile := filepath.Join(stageDir, "artifact")
	httpClient, err := buildHTTPClient(cfg, log)
	if err != nil {
		return Result{}, err
	}
	hashOut, err := fetch.DownloadWithClient(ctx, httpClient, cfg.ArtifactURL, artifactFile)
	if err != nil {
		return Result{}, err
	}
	if !strings.EqualFold(hashOut, cfg.SHA256) {
		return Result{}, fmt.Errorf("artifact sha256 mismatch: expected %s, got %s", cfg.SHA256, hashOut)
	}

	extractedDir := filepath.Join(stageDir, "extract")
	if err := os.MkdirAll(extractedDir, 0750); err != nil {
		return Result{}, err
	}
	if err := extractArtifact(artifactFile, cfg.ArtifactURL, extractedDir); err != nil {
		return Result{}, err
	}

	binPath, err := locateBinPath(extractedDir, runtime.GOOS)
	if err != nil {
		return Result{}, err
	}
	version, err := install.VerifyOsqueryBinary(runtime.GOOS, binPath, log)
	if err != nil {
		return Result{}, err
	}

	if err := os.MkdirAll(filepath.Dir(releaseDir), 0750); err != nil {
		return Result{}, err
	}
	if _, statErr := os.Stat(releaseDir); statErr == nil {
		if res, ok := tryReuseInstalled(releaseDir, cfg, log); ok {
			_ = writeActiveReleaseFile(installDir, releaseDir)
			return res, nil
		}
		_ = os.RemoveAll(releaseDir)
	}
	if err := os.Rename(extractedDir, releaseDir); err != nil {
		return Result{}, err
	}

	relBinPath, err := filepath.Rel(extractedDir, binPath)
	if err != nil {
		return Result{}, err
	}

	meta := metadata{
		ArtifactURL: cfg.ArtifactURL,
		SHA256:      strings.ToLower(cfg.SHA256),
		Version:     version,
		InstalledAt: time.Now().UTC(),
	}
	if err := writeMetadata(filepath.Join(releaseDir, releaseMetadataFile), meta); err != nil {
		return Result{}, err
	}
	if err := writeActiveReleaseFile(installDir, releaseDir); err != nil {
		return Result{}, err
	}
	if err := cleanupOldReleases(installDir, releaseDir); err != nil {
		return Result{}, err
	}

	return Result{
		BinPath: filepath.Join(releaseDir, relBinPath),
		Version: version,
	}, nil
}

func ResolveInstallDir(dataPath, cfgInstallDir string) string {
	if strings.TrimSpace(cfgInstallDir) != "" {
		return cfgInstallDir
	}
	return filepath.Join(dataPath, "osquery-install")
}

// RemoveInstalled removes previously managed custom osquery artifact state.
// It is used when custom artifact config is removed and osquerybeat should
// return to bundled-only mode.
func RemoveInstalled(installDir string) error {
	releasesDir := filepath.Join(installDir, releasesDirName)
	if err := os.RemoveAll(releasesDir); err != nil {
		return fmt.Errorf("failed removing releases directory %s: %w", releasesDir, err)
	}

	activeReleasePath := filepath.Join(installDir, activeReleaseFile)
	if err := os.Remove(activeReleasePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed removing active release marker %s: %w", activeReleasePath, err)
	}
	return nil
}

func tryReuseInstalled(releaseDir string, cfg config.InstallConfig, log *logp.Logger) (Result, bool) {
	metaPath := filepath.Join(releaseDir, releaseMetadataFile)
	meta, err := readMetadata(metaPath)
	if err != nil {
		log.Debugf("custom osquery metadata not found in %s: %v", releaseDir, err)
		return Result{}, false
	}
	if !strings.EqualFold(meta.SHA256, cfg.SHA256) {
		return Result{}, false
	}

	binPath, err := locateBinPath(releaseDir, runtime.GOOS)
	if err != nil {
		return Result{}, false
	}
	version, err := install.VerifyOsqueryBinary(runtime.GOOS, binPath, log)
	if err != nil {
		return Result{}, false
	}
	return Result{BinPath: binPath, Version: version}, true
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

func writeActiveReleaseFile(installDir, releaseDir string) error {
	fp := filepath.Join(installDir, activeReleaseFile)
	tmp := fp + ".tmp"
	if err := os.WriteFile(tmp, []byte(releaseDir), 0640); err != nil {
		return err
	}
	return os.Rename(tmp, fp)
}

func buildHTTPClient(cfg config.InstallConfig, log *logp.Logger) (*http.Client, error) {
	client := &http.Client{}
	if cfg.SSL == nil {
		return client, nil
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	sslCfg, err := tlscommon.LoadTLSConfig(cfg.SSL, log)
	if err != nil {
		return nil, fmt.Errorf("failed loading osquery.elastic_options.install.ssl config: %w", err)
	}
	transport.TLSClientConfig = sslCfg.ToConfig()
	client.Transport = transport
	return client, nil
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
	lower := strings.ToLower(artifactURL)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return tar.ExtractFile(artifactFile, destinationDir)
	case strings.HasSuffix(lower, ".zip"):
		return zip.UnzipFile(artifactFile, destinationDir)
	case strings.HasSuffix(lower, ".pkg"):
		return pkgutil.Expand(artifactFile, destinationDir)
	case strings.HasSuffix(lower, ".msi"):
		return msiutil.Expand(artifactFile, destinationDir)
	default:
		return fmt.Errorf("unsupported artifact format for %q", artifactURL)
	}
}

func locateBinPath(root, goos string) (string, error) {
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

		candidates = append(candidates, filepath.Dir(path))
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
