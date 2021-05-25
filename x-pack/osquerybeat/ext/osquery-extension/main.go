// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Borrowed from https://github.com/kolide/launcher/blob/master/cmd/osquery-extension/osquery-extension.go
// Original license from the kolide launcher repository

// MIT License

// Copyright (c) 2017 Kolide

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"
)

func main() {
	var (
		_ = flag.Bool("verbose", false, "")
		_ = flag.Int("interval", 0, "")
		_ = flag.Int("timeout", 0, "")
		_ = flag.String("socket", "", "")
	)
	flag.Parse()

	fmt.Fprintf(os.Stderr, "%+v", os.Args)

	go monitorForParent()

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)
	<-sig
}

// continuously monitor for ppid and exit if osqueryd is no longer the parent process.
// because osqueryd is always the process starting the extension, when osqueryd is killed this process should also be cleaned up.
// sometimes the termination is not clean, causing this process to remain running, which sometimes prevents osqueryd from properly restarting.
// https://github.com/kolide/launcher/issues/341
func monitorForParent() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	f := func() {
		ppid := os.Getppid()
		if ppid <= 1 {
			fmt.Println("extension process no longer owned by osqueryd, quitting")
			os.Exit(1)
		}
	}

	f()

	select {
	case <-ticker.C:
		f()
	}
}
