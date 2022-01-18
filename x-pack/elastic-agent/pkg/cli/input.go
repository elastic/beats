// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
)

// ReadInput shows the text and ask the user to provide input.
func ReadInput(prompt string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	return input(reader, os.Stdout, prompt)
}

func input(r io.Reader, out io.Writer, prompt string) (string, error) {
	reader := bufio.NewScanner(r)
	fmt.Fprintf(out, prompt+" ")

	if !reader.Scan() {
		return "", errors.New("error reading user input")
	}
	return reader.Text(), nil
}
