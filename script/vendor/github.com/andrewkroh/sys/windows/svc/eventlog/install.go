// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package eventlog

import (
	"fmt"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	// Log levels.
	Success      = windows.EVENTLOG_SUCCESS
	Info         = windows.EVENTLOG_INFORMATION_TYPE
	Warning      = windows.EVENTLOG_WARNING_TYPE
	Error        = windows.EVENTLOG_ERROR_TYPE
	AuditSuccess = windows.EVENTLOG_AUDIT_SUCCESS
	AuditFailure = windows.EVENTLOG_AUDIT_FAILURE
)

// Application event log provider.
const Application = "Application"

const eventLogKeyName = `SYSTEM\CurrentControlSet\Services\EventLog`

// Install modifies PC registry to allow logging with an event source src.
// It adds all required keys and values to the event log registry key.
// Install uses msgFile as the event message file. If useExpandKey is true,
// the event message file is installed as REG_EXPAND_SZ value,
// otherwise as REG_SZ. Use bitwise of log.Error, log.Warning and
// log.Info to specify events supported by the new event source.
func Install(provider, src, msgFile string, useExpandKey bool, eventsSupported uint32) (bool, error) {
	eventLogKey, err := registry.OpenKey(registry.LOCAL_MACHINE, eventLogKeyName, registry.CREATE_SUB_KEY)
	if err != nil {
		return false, err
	}
	defer eventLogKey.Close()

	pk, _, err := registry.CreateKey(eventLogKey, provider, registry.SET_VALUE)
	if err != nil {
		return false, err
	}
	defer pk.Close()

	sk, alreadyExist, err := registry.CreateKey(pk, src, registry.SET_VALUE)
	if err != nil {
		return false, err
	}
	defer sk.Close()
	if alreadyExist {
		return true, nil
	}

	err = sk.SetDWordValue("CustomSource", 1)
	if err != nil {
		return false, err
	}
	if useExpandKey {
		err = sk.SetExpandStringValue("EventMessageFile", msgFile)
	} else {
		err = sk.SetStringValue("EventMessageFile", msgFile)
	}
	if err != nil {
		return false, err
	}
	err = sk.SetDWordValue("TypesSupported", eventsSupported)
	if err != nil {
		return false, err
	}
	return false, nil
}

// InstallAsEventCreate is the same as Install, but uses
// %SystemRoot%\System32\EventCreate.exe as the event message file.
func InstallAsEventCreate(provider, src string, eventsSupported uint32) (bool, error) {
	alreadyExists, err := Install(provider, src, "%SystemRoot%\\System32\\EventCreate.exe", true, eventsSupported)
	return alreadyExists, err
}

// Remove deletes all registry elements installed for an event logging source.
func RemoveSource(provider, src string) error {
	providerKeyName := fmt.Sprintf("%s\\%s", eventLogKeyName, provider)
	pk, err := registry.OpenKey(registry.LOCAL_MACHINE, providerKeyName, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer pk.Close()
	return registry.DeleteKey(pk, src)
}

// Remove deletes all registry elements installed for an event logging provider.
// Only use this method if you have installed a custom provider.
func RemoveProvider(provider string) error {
	// Protect against removing Application.
	if provider == Application {
		return fmt.Errorf("%s cannot be removed. Only custom providers can be removed.")
	}

	eventLogKey, err := registry.OpenKey(registry.LOCAL_MACHINE, eventLogKeyName, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer eventLogKey.Close()
	return registry.DeleteKey(eventLogKey, provider)
}
