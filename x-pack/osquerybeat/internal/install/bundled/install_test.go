// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package bundled

import (
	"testing"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/distro"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/install/artifactformat"
)

func TestBuildPlan(t *testing.T) {
	installDir := t.TempDir()

	tests := []struct {
		name    string
		osarch  distro.OSArch
		format  artifactformat.Format
		wantErr bool
	}{
		{name: "linux tar", osarch: distro.OSArch{OS: "linux", Arch: "amd64"}, format: artifactformat.TarGz},
		{name: "darwin pkg", osarch: distro.OSArch{OS: "darwin", Arch: "arm64"}, format: artifactformat.Pkg},
		{name: "windows msi", osarch: distro.OSArch{OS: "windows", Arch: "amd64"}, format: artifactformat.Msi},
		{name: "windows zip", osarch: distro.OSArch{OS: "windows", Arch: "arm64"}, format: artifactformat.Zip},
		{name: "unsupported", osarch: distro.OSArch{OS: "linux", Arch: "amd64"}, format: artifactformat.Unknown, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := BuildPlan(tc.osarch, tc.format, installDir)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if plan.OSQuerySourcePath == "" || plan.CertsSourcePath == "" || plan.OSQueryTargetPath == "" {
				t.Fatalf("plan has empty paths: %+v", plan)
			}
		})
	}
}
