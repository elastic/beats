// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin || synthetics

package synthexec

import (
	"fmt"
	"os/exec"
	"strings"
)

// Variant of exec.command with redacted params and playwright options,
// which might contain sensitive information.
type SynthCmd struct {
	*exec.Cmd
}

func (cmd *SynthCmd) String() string {
	b := new(strings.Builder)
	b.WriteString(cmd.Path)
	for i := 1; i < len(cmd.Args); i++ {
		b.WriteString(" ")
		a := cmd.Args[i]
		switch a {
		case "--params":
			fallthrough
		case "--playwright-options":
			b.WriteString(fmt.Sprintf("%s { REDACTED }", a))
			i++
		default:
			b.WriteString(a)
		}
	}

	return b.String()
}

// Formatter override redacting params
func (cmd SynthCmd) Format(f fmt.State, verb rune) {

	f.Write([]byte(cmd.String()))
}
