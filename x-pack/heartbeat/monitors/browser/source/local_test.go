// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package source

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalSourceValidate(t *testing.T) {
	tests := []struct {
		name     string
		OrigPath string
		err      error
	}{
		{"valid", "./", nil},
		{"invalid", "/not/a/path", ErrInvalidPath("/not/a/path")},
		{"nopath", "", ErrNoPath},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LocalSource{OrigPath: tt.OrigPath}
			err := l.Validate()
			if tt.err == nil {
				require.NoError(t, err)
			} else {
				require.Regexp(t, tt.err, err)
			}
		})
	}
}
