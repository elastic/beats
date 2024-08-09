// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"os"

	"kernel.org/pub/linux/libs/security/libcap/cap"
)

type capProc interface {
	GetFlag(vec cap.Flag, val cap.Value) (bool, error)
	SetFlag(vec cap.Flag, enable bool, val ...cap.Value) error
	SetProc() error
}

var (
	// for unit-testing
	capProcFunc = func() capProc {
		return cap.GetProc()
	}
)

func initCapabilities() {
	isRoot, err := hasRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}
	if !isRoot {
		if err := raiseEffectiveCapabilities(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}
}

// raiseEffectiveCapabilities raises the capabilities of the Effective and Inheritable sets to match
// the ones in the Permitted set. Note that any capabilities that are not part of the Bounding set
// are exclude by the OS from the Permitted set.
func raiseEffectiveCapabilities() error {
	procCaps := capProcFunc()

	setProc := false

	for val := cap.Value(0); val < cap.MaxBits(); val++ {
		permittedHasCap, err := procCaps.GetFlag(cap.Permitted, val)
		if err != nil {
			return fmt.Errorf("get cap from permitted failed: %w", err)
		}
		if !permittedHasCap {
			continue
		}

		effectiveHasCap, err := procCaps.GetFlag(cap.Effective, val)
		if err != nil {
			return fmt.Errorf("get cap from effective failed: %w", err)
		}
		if !effectiveHasCap {
			err = procCaps.SetFlag(cap.Effective, true, val)
			if err != nil {
				return fmt.Errorf("set cap to permitted failed: %w", err)
			}
			setProc = true
		}

		inheritableHasCap, err := procCaps.GetFlag(cap.Inheritable, val)
		if err != nil {
			return fmt.Errorf("get cap from effective failed: %w", err)
		}
		if !inheritableHasCap {
			err = procCaps.SetFlag(cap.Inheritable, true, val)
			if err != nil {
				return fmt.Errorf("set cap to inheritable failed: %w", err)
			}
			setProc = true
		}
	}

	if !setProc {
		return nil
	}

	if err := procCaps.SetProc(); err != nil {
		return fmt.Errorf("set proc failed: %w", err)
	}

	return nil
}

// hasRoot returns true if the user has root permissions.
// Added extra `nil` value to return since the hasRoot for windows will return an error as well
func hasRoot() (bool, error) {
	return os.Geteuid() == 0, nil
}
