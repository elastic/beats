// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package benchmark

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := map[string]struct {
		cfg         benchmarkConfig
		expectError bool
		errorString string
	}{
		"default":     {cfg: defaultConfig},
		"countAndEps": {cfg: benchmarkConfig{Message: "a", Count: 1, Eps: 1}, expectError: true, errorString: "only one of count or eps may be specified"},
		"empty":       {cfg: benchmarkConfig{}, expectError: true, errorString: "message must be specified"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if err == nil && tc.expectError == true {
				t.Fatalf("expected validation error, didn't get it")
			}
			if err != nil && tc.expectError == false {
				t.Fatalf("unexpected validation error: %s", err)
			}
			if err != nil && !strings.Contains(err.Error(), tc.errorString) {
				t.Fatalf("error: '%s' didn't contain expected string: '%s'", err, tc.errorString)
			}
		})
	}
}
