// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import "testing"

func TestInstallConfigValidate(t *testing.T) {
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
				ArtifactURL: "https://example.com/osquery.tar.gz",
			},
			wantErr: true,
		},
		{
			name: "invalid checksum",
			cfg: InstallConfig{
				ArtifactURL: "https://example.com/osquery.tar.gz",
				SHA256:      "abc",
			},
			wantErr: true,
		},
		{
			name: "reject non https by default",
			cfg: InstallConfig{
				ArtifactURL: "http://example.com/osquery.tar.gz",
				SHA256:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
			wantErr: true,
		},
		{
			name: "reject custom install_dir",
			cfg: InstallConfig{
				ArtifactURL: "https://example.com/osquery.tar.gz",
				SHA256:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				InstallDir:  "/tmp/custom",
			},
			wantErr: true,
		},
		{
			name: "allow insecure URL override",
			cfg: InstallConfig{
				ArtifactURL:      "http://example.com/osquery.tar.gz",
				SHA256:           "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				AllowInsecureURL: true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
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
			ArtifactURL: "https://example.org/osquery.tar.gz",
			SHA256:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
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
		if cfg.ArtifactURL != installCfg.ArtifactURL {
			t.Fatalf("unexpected artifact_url: %s", cfg.ArtifactURL)
		}
	})
}
