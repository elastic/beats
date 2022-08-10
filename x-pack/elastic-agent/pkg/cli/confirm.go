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
	"strings"
)

// Confirm shows the confirmation text and ask the user to answer (y/n)
// default will be shown in uppercase and be selected if the user hits enter
// returns true for yes, false for no
func Confirm(prompt string, def bool) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	return confirm(reader, os.Stdout, prompt, def)
}

func confirm(r io.Reader, out io.Writer, prompt string, def bool) (bool, error) {
	options := " [Y/n]"
	if !def {
		options = " [y/N]"
	}

	reader := bufio.NewScanner(r)
	for {
		fmt.Fprintf(out, prompt+options+":")

		if !reader.Scan() {
			break
		}
		switch strings.ToLower(reader.Text()) {
		case "":
			return def, nil
		case "y", "yes", "yeah":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			fmt.Fprintln(out, "Please write 'y' or 'n'")
		}
	}

	return false, errors.New("error reading user input")
}
