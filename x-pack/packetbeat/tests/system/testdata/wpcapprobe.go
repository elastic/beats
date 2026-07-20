// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows"

	// Import the npcap package like packetbeat does to verify it does not load
	// wpcap.dll automatically
	_ "github.com/elastic/beats/v7/packetbeat/npcap"
)

func main() {
	name, err := windows.UTF16PtrFromString("wpcap.dll")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(65)
	}

	var h windows.Handle
	err = windows.GetModuleHandleEx(windows.GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT, name, &h)
	if err != nil {
		if errors.Is(err, windows.ERROR_MOD_NOT_FOUND) {
			return // not loaded: the expected lazy behavior
		}
		fmt.Fprintln(os.Stderr, "GetModuleHandleEx:", err)
		os.Exit(65)
	}
	if h != 0 {
		fmt.Fprintln(os.Stderr, "wpcap.dll is mapped into a process that only imports the capture code")
		os.Exit(1)
	}
}
