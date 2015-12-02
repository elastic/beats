// Copyright 2012 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package eventlog_test

import (
	"testing"

	"github.com/andrewkroh/sys/windows/svc/eventlog"
)

func TestLog(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode - it modifies system logs")
	}

	const name = "mylog"
	const supports = eventlog.Error | eventlog.Warning | eventlog.Info |
		eventlog.Success | eventlog.AuditSuccess | eventlog.AuditFailure
	alreadyExists, err := eventlog.InstallAsEventCreate(eventlog.Application, name, supports)
	if err != nil {
		t.Fatalf("Install failed: %s", err)
	}
	t.Log("Already exists:", alreadyExists)
	defer func() {
		err = eventlog.RemoveSource(eventlog.Application, name)
		if err != nil {
			t.Fatalf("Remove failed: %s", err)
		}
	}()

	l, err := eventlog.Open(name)
	if err != nil {
		t.Fatalf("Open failed: %s", err)
	}
	defer l.Close()

	err = l.Success(1, "success")
	if err != nil {
		t.Fatalf("Successo failed: %s", err)
	}
	err = l.Info(2, "info")
	if err != nil {
		t.Fatalf("Info failed: %s", err)
	}
	err = l.Warning(3, "warning")
	if err != nil {
		t.Fatalf("Warning failed: %s", err)
	}
	err = l.Error(4, "error")
	if err != nil {
		t.Fatalf("Error failed: %s", err)
	}
	err = l.AuditSuccess(5, "audit success")
	if err != nil {
		t.Fatalf("AuditSuccess failed: %s", err)
	}
	err = l.AuditFailure(6, "audit failure")
	if err != nil {
		t.Fatalf("AuditFailure failed: %s", err)
	}
}
