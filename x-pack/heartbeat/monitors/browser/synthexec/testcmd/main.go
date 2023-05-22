// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux || darwin

package main

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

func main() {
	//Sleep first to test timeout feature async
	time.Sleep(time.Millisecond * 500)
	// For sending JSON results
	pipe := os.NewFile(3, "pipe")

	// Exit immediately to test this error condition
	if len(os.Args) > 1 && os.Args[1] == "exit" {
		os.Exit(123)
	}

	file, err := os.Open("sample.ndjson")
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not open samplerun.ndjson: %s\n", err)
		os.Exit(1)
	}
	scanner := bufio.NewScanner(file)
	i := 0
	for scanner.Scan() {
		// We need to test console out within a journey context
		// so we wait till the first line, a journey/start is written
		// we need to make sure the these raw lines are received after
		// the journey start, so, even though we're careful to use
		// un-buffered I/O we sleep for a generous 100ms before and after
		// to make sure these lines are in the right context
		// otherwise they might get lost.
		// Note, in the real world lost lines here aren't a big deal
		// these only are relevant in error situations, and this is a
		// pathological case
		if i == 1 {
			time.Sleep(time.Millisecond * 100)
			stdin := bufio.NewReader(os.Stdin)
			stdinLine, _ := stdin.ReadString('\n')
			os.Stdout.WriteString(stdinLine + "\n")
			os.Stderr.WriteString("Stderr 1\n")
			os.Stderr.WriteString("Stderr 2\n")
			time.Sleep(time.Millisecond * 100)
		}
		_, _ = pipe.WriteString(scanner.Text())
		_, _ = pipe.WriteString("\n")
		i++
	}
	if scanner.Err() != nil {
		//nolint:forbidigo // we don't care about this test command
		fmt.Printf("Scanner error %s", scanner.Err())
		os.Exit(1)
	}
}
