// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const githubReleasesAPI = "https://api.github.com/repos/osquery/osquery/releases"

type releaseAsset struct {
	Name               string `json:"name"`
	Digest             string `json:"digest"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type release struct {
	TagName string         `json:"tag_name"`
	Draft   bool           `json:"draft"`
	Assets  []releaseAsset `json:"assets"`
}

var artifactConstants = map[string]string{
	"osquery-%s.pkg":                    "osqueryDistroDarwinSHA256",
	"osquery-%s_1.linux_x86_64.tar.gz":  "osqueryDistroLinuxSHA256",
	"osquery-%s_1.linux_aarch64.tar.gz": "osqueryDistroLinuxARMSHA256",
	"osquery-%s.windows_arm64.zip":      "osqueryDistroWindowsARMZipSHA256",
	"osquery-%s.windows_x86_64.zip":     "osqueryDistroWindowsX86ZipSHA256",
}

func main() {
	version := flag.String("version", "latest", "Osquery release version, or latest")
	distroFile := flag.String("distro-file", "internal/distro/distro.go", "Path to distro.go")
	changelogDir := flag.String("changelog-dir", "../../changelog/fragments", "Path to changelog fragments")
	flag.Parse()

	rel, err := fetchRelease(http.DefaultClient, githubReleasesAPI, *version)
	if err != nil {
		fatal(err)
	}
	resolvedVersion := strings.TrimPrefix(rel.TagName, "v")
	hashes, err := releaseHashes(http.DefaultClient, rel, resolvedVersion)
	if err != nil {
		fatal(err)
	}
	changed, err := updateDistroFile(*distroFile, resolvedVersion, hashes)
	if err != nil {
		fatal(err)
	}
	if changed {
		if err := ensureChangelogFragment(*changelogDir, resolvedVersion, time.Now()); err != nil {
			fatal(err)
		}
		fmt.Printf("Updated bundled Osquery metadata to %s.\n", resolvedVersion)
		return
	}
	fmt.Printf("Bundled Osquery metadata is already at %s.\n", resolvedVersion)
}

func fetchRelease(client *http.Client, apiBase, requested string) (release, error) {
	requested = strings.TrimSpace(requested)
	url := strings.TrimRight(apiBase, "/") + "/latest"
	if requested != "" && !strings.EqualFold(requested, "latest") {
		url = strings.TrimRight(apiBase, "/") + "/tags/" + strings.TrimPrefix(requested, "v")
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return release{}, fmt.Errorf("GitHub release request failed: %s", resp.Status)
	}
	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return release{}, err
	}
	if rel.Draft || strings.TrimSpace(rel.TagName) == "" {
		return release{}, errors.New("GitHub returned an invalid or draft release")
	}
	return rel, nil
}

func releaseHashes(client *http.Client, rel release, version string) (map[string]string, error) {
	assets := make(map[string]releaseAsset, len(rel.Assets))
	for _, asset := range rel.Assets {
		assets[asset.Name] = asset
	}
	hashes := make(map[string]string, len(artifactConstants))
	for pattern, constantName := range artifactConstants {
		name := fmt.Sprintf(pattern, version)
		asset, ok := assets[name]
		if !ok {
			return nil, fmt.Errorf("release %s does not contain %s", version, name)
		}
		digest := strings.TrimPrefix(strings.TrimSpace(asset.Digest), "sha256:")
		if decoded, err := hex.DecodeString(digest); err != nil || len(decoded) != sha256.Size {
			if sidecar, ok := assets[name+".sha256"]; ok {
				var err error
				digest, err = fetchSidecarHash(client, sidecar.BrowserDownloadURL)
				if err != nil {
					return nil, fmt.Errorf("fetch sidecar hash for %s: %w", name, err)
				}
			} else {
				var err error
				digest, err = downloadSHA256(client, asset.BrowserDownloadURL)
				if err != nil {
					return nil, fmt.Errorf("calculate digest for %s: %w", name, err)
				}
			}
		}
		hashes[constantName] = digest
	}
	return hashes, nil
}

// fetchSidecarHash downloads a .sha256 sidecar file and returns the hex digest.
// Sidecar files contain "hash  filename" (shasum format) or just "hash".
func fetchSidecarHash(client *http.Client, url string) (string, error) {
	if strings.TrimSpace(url) == "" {
		return "", errors.New("sidecar asset has no download URL")
	}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	hash := strings.Fields(string(b))
	if len(hash) == 0 {
		return "", errors.New("sidecar file is empty")
	}
	digest := hash[0]
	if decoded, err := hex.DecodeString(digest); err != nil || len(decoded) != sha256.Size {
		return "", fmt.Errorf("sidecar file does not contain a valid SHA256: %q", digest)
	}
	return digest, nil
}

func downloadSHA256(client *http.Client, url string) (string, error) {
	if strings.TrimSpace(url) == "" {
		return "", errors.New("asset has no digest or download URL")
	}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}
	h := sha256.New()
	if _, err := io.Copy(h, resp.Body); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func updateDistroFile(path, version string, hashes map[string]string) (bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	updated := string(b)
	values := make(map[string]string, len(hashes)+1)
	values["osqueryVersion"] = version
	for name, hash := range hashes {
		values[name] = hash
	}
	for name, value := range values {
		re := regexp.MustCompile(`(?m)^(\s*` + regexp.QuoteMeta(name) + `\s*=\s*)"[^"]+"`)
		if !re.MatchString(updated) {
			return false, fmt.Errorf("constant %s not found in %s", name, path)
		}
		updated = re.ReplaceAllString(updated, `${1}"`+value+`"`)
	}
	if updated == string(b) {
		return false, nil
	}
	return true, os.WriteFile(path, []byte(updated), 0o644)
}

func ensureChangelogFragment(dir, version string, now time.Time) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	summary := "upgrade osquery version to " + version
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return err
		}
		if strings.Contains(string(b), "summary: "+summary) {
			return nil
		}
	}
	content := fmt.Sprintf("kind: upgrade\n\nsummary: %s\n\ndescription: Upgrade the bundled osquery runtime to version %s.\n\ncomponent: osquerybeat\n", summary, version)
	name := fmt.Sprintf("%d-upgrade-osquery-%s.yaml", now.Unix(), version)
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
