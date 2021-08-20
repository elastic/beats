// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	noOutput := flag.Bool("no-output", false, "disable output")
	output := flag.String("output", "stderr", "output destination")
	exitcode := flag.Int("exitcode", 0, "exit code")
	flag.Parse()

	if *noOutput {
		os.Exit(*exitcode)
	}

	var dest io.Writer
	if *output == "stdout" {
		dest = os.Stdout
	} else if *output == "stderr" {
		dest = os.Stderr
	} else {
		panic("unknown destination")
	}

	fmt.Fprintf(dest, "message written to %s", *output)
	os.Exit(*exitcode)
}
