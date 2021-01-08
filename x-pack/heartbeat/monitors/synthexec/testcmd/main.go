// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	// Sample output to test stdout
	stdin := bufio.NewReader(os.Stdin)

	stdinLine, _ := stdin.ReadString('\n')
	fmt.Fprintln(os.Stdout, stdinLine)
	fmt.Fprintln(os.Stderr, "Stderr 1")
	fmt.Fprintln(os.Stderr, "Stderr 2")

	// For sending JSON results
	pipe := os.NewFile(3, "pipe")

	file, err := os.Open("sample.ndjson")
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not open samplerun.ndjson: %s\n", err)
		os.Exit(1)
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fmt.Fprintln(pipe, scanner.Text())
	}
	if scanner.Err() != nil {
		fmt.Printf("Scanner error %s", scanner.Err())
		os.Exit(1)
	}
}
