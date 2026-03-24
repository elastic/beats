// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package artifactformat

import "testing"

func TestDetect(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      Format
		expectErr bool
	}{
		{name: "tar gz", input: "https://example.org/osquery.tar.gz", want: TarGz},
		{name: "tgz", input: "https://example.org/osquery.tgz", want: TarGz},
		{name: "zip", input: "https://example.org/osquery.zip", want: Zip},
		{name: "pkg", input: "https://example.org/osquery.pkg", want: Pkg},
		{name: "msi", input: "https://example.org/osquery.msi", want: Msi},
		{name: "signed query string", input: "https://example.org/osquery.tar.gz?X-Amz-Signature=abc", want: TarGz},
		{name: "unsupported", input: "https://example.org/osquery.bin", expectErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Detect(tc.input)
			if tc.expectErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %s, got %s", tc.want, got)
			}
		})
	}
}
