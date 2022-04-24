// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

func main() {
	// For sending JSON results
	pipe := os.NewFile(3, "pipe")

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

		//nolint:errcheck // There are no new changes to this line but
		// linter has been activated in the meantime. We'll cleanup separately.
		pipe.WriteString(scanner.Text())
		//nolint:errcheck // There are no new changes to this line but
		// linter has been activated in the meantime. We'll cleanup separately.
		pipe.WriteString("\n")
		i++
	}
	if scanner.Err() != nil {
		//nolint:forbidigo // There are no new changes to this line but
		// linter has been activated in the meantime. We'll cleanup separately.
		fmt.Printf("Scanner error %s", scanner.Err())
		os.Exit(1)
	}
}
