// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build windows

package cmd

import (
	"os"
	"strings"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

func createSymlink(oldPath, newPath string, argsOverrides ...string) error {
	args := strings.Join(argsOverrides, " ")
	linkPath := oldPath + ".lnk"
	ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED|ole.COINIT_SPEED_OVER_MEMORY)
	oleShellObject, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return err
	}
	defer oleShellObject.Release()

	wshell, err := oleShellObject.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer wshell.Release()

	cs, err := oleutil.CallMethod(wshell, "CreateShortcut", linkPath)
	if err != nil {
		return err
	}

	idispatch := cs.ToIDispatch()
	if _, err := oleutil.CallMethod(idispatch, "Save"); err != nil {
		return err
	}

	oleutil.PutProperty(idispatch, "TargetPath", newPath)
	if _, err := oleutil.CallMethod(idispatch, "Save"); err != nil {
		return err
	}

	return os.Symlink(linkPath, oldPath)
}
