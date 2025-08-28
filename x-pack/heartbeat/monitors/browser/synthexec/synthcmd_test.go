// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux || synthetics

package synthexec

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSynthCmdStringOutput(t *testing.T) {
	tests := []struct {
		name     string
		stringer func(cmd SynthCmd) string
	}{
		{
			name: "fmt.Sprintf",
			stringer: func(cmd SynthCmd) string {
				return fmt.Sprintf("%s", cmd)
			},
		},
		{
			name: "fmt.Println",
			stringer: func(cmd SynthCmd) string {
				r, w, err := os.Pipe()
				assert.NoError(t, err)
				fmt.Fprint(w, cmd)
				w.Close()
				defer r.Close()

				o, err := io.ReadAll(r)
				assert.NoError(t, err)

				return string(o)
			},
		},
		{
			name: "cmd.String()",
			stringer: func(cmd SynthCmd) string {
				return cmd.String()
			},
		},
	}

	redacted := []string{"secret", "mysecrettoken", "auth", "mysecretauth"}
	cmd := SynthCmd{
		exec.Command("/nil", "--params", "{'secret':'mysecrettoken'}", "--playwright-options", "{'auth':'mysecretauth'}"),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.stringer(cmd)
			for _, r := range redacted {
				assert.NotContains(t, s, r)
			}
		})
	}
}
