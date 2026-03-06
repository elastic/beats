// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"testing"

	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

func boolPtr(v bool) *bool { return &v }

func TestInstallConfigNormalizeAndValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     InstallConfig
		wantErr bool
	}{
		{
			name: "disabled is valid",
			cfg:  InstallConfig{},
		},
		{
			name: "artifact requires checksum",
			cfg: InstallConfig{
				Linux: &InstallArtifactConfig{
					ArtifactURL: "https://example.com/osquery.tar.gz",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid checksum",
			cfg: InstallConfig{
				Linux: &InstallArtifactConfig{
					ArtifactURL: "https://example.com/osquery.tar.gz",
					SHA256:      "abc",
				},
			},
			wantErr: true,
		},
		{
			name: "reject non https by default",
			cfg: InstallConfig{
				Linux: &InstallArtifactConfig{
					ArtifactURL: "http://example.com/osquery.tar.gz",
					SHA256:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
			},
			wantErr: true,
		},
		{
			name: "allow insecure URL override",
			cfg: InstallConfig{
				Linux: &InstallArtifactConfig{
					ArtifactURL: "http://example.com/osquery.tar.gz",
					SHA256:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
				AllowInsecureURL: true,
			},
		},
		{
			name: "allow insecure URL override per-platform",
			cfg: InstallConfig{
				Linux: &InstallArtifactConfig{
					ArtifactURL:      "http://example.com/osquery.tar.gz",
					SHA256:           "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					AllowInsecureURL: boolPtr(true),
				},
			},
		},
		{
			name: "deny insecure URL override per-platform",
			cfg: InstallConfig{
				Linux: &InstallArtifactConfig{
					ArtifactURL:      "http://example.com/osquery.tar.gz",
					SHA256:           "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					AllowInsecureURL: boolPtr(false),
				},
				AllowInsecureURL: true,
			},
			wantErr: true,
		},
		{
			name: "multiple platforms valid",
			cfg: InstallConfig{
				Linux: &InstallArtifactConfig{
					ArtifactURL: "https://example.com/osquery-linux.tar.gz",
					SHA256:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
				Darwin: &InstallArtifactConfig{
					ArtifactURL: "https://example.com/osquery-darwin.pkg",
					SHA256:      "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				},
				Windows: &InstallArtifactConfig{
					ArtifactURL: "https://example.com/osquery-windows.msi",
					SHA256:      "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
				},
			},
		},
		{
			name: "sha requires artifact url",
			cfg: InstallConfig{
				Linux: &InstallArtifactConfig{
					SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
			},
			wantErr: true,
		},
		{
			name: "normalizes sha to lowercase",
			cfg: InstallConfig{
				Linux: &InstallArtifactConfig{
					ArtifactURL: "https://example.com/osquery.tar.gz",
					SHA256:      "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.NormalizeAndValidate()
			if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.name == "normalizes sha to lowercase" {
				if tc.cfg.Linux.SHA256 != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
					t.Fatalf("expected lowercased sha256, got %s", tc.cfg.Linux.SHA256)
				}
			}
		})
	}
}

func TestInstallConfigPlatformSelection(t *testing.T) {
	cfg := InstallConfig{
		Linux: &InstallArtifactConfig{
			ArtifactURL: "https://example.org/osquery-linux.tar.gz",
			SHA256:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		Darwin: &InstallArtifactConfig{
			ArtifactURL: "https://example.org/osquery-darwin.pkg",
			SHA256:      "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
	}

	if !cfg.Enabled() {
		t.Fatal("expected config to be enabled")
	}
	if !cfg.EnabledForPlatform("linux") {
		t.Fatal("expected linux to be enabled")
	}
	if cfg.EnabledForPlatform("windows") {
		t.Fatal("expected windows to be disabled")
	}

	selected, ok := cfg.SelectedForPlatform("darwin")
	if !ok {
		t.Fatal("expected darwin selection")
	}
	if selected.ArtifactURL != cfg.Darwin.ArtifactURL {
		t.Fatalf("unexpected selected darwin artifact url: %s", selected.ArtifactURL)
	}

	if _, ok := cfg.SelectedForPlatform("windows"); ok {
		t.Fatal("expected no windows selection")
	}
}

func TestInstallConfigPlatformOverrides(t *testing.T) {
	globalSSL := &tlscommon.Config{}
	linuxSSL := &tlscommon.Config{}
	cfg := InstallConfig{
		AllowInsecureURL: false,
		SSL:              globalSSL,
		Linux: &InstallArtifactConfig{
			ArtifactURL:      "https://example.org/osquery-linux.tar.gz",
			SHA256:           "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			AllowInsecureURL: boolPtr(true),
			SSL:              linuxSSL,
		},
	}

	if !cfg.AllowInsecureURLForPlatform("linux") {
		t.Fatal("expected linux allow_insecure_url override to be true")
	}
	if cfg.AllowInsecureURLForPlatform("windows") {
		t.Fatal("expected windows allow_insecure_url to use top-level false")
	}
	if cfg.SSLForPlatform("linux") != linuxSSL {
		t.Fatal("expected linux ssl override to be used")
	}
	if cfg.SSLForPlatform("windows") != globalSSL {
		t.Fatal("expected windows ssl to use top-level config")
	}
}

func TestGetOsqueryInstallConfig(t *testing.T) {
	t.Run("missing input returns empty", func(t *testing.T) {
		cfg := GetOsqueryInstallConfig(nil)
		if cfg.Enabled() {
			t.Fatal("expected disabled install config")
		}
	})

	t.Run("returns first input osquery install", func(t *testing.T) {
		installCfg := &InstallConfig{
			Linux: &InstallArtifactConfig{
				ArtifactURL: "https://example.org/osquery-linux.tar.gz",
				SHA256:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
		}
		inputs := []InputConfig{
			{
				Osquery: &OsqueryConfig{
					ElasticOptions: &ElasticOptions{
						Install: installCfg,
					},
				},
			},
		}
		cfg := GetOsqueryInstallConfig(inputs)
		selected, ok := cfg.SelectedForPlatform("linux")
		if !ok {
			t.Fatalf("expected linux config")
		}
		if selected.ArtifactURL != installCfg.Linux.ArtifactURL {
			t.Fatalf("unexpected artifact_url: %s", selected.ArtifactURL)
		}
	})
}
