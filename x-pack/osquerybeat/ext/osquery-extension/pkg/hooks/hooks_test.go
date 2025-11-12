// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hooks

import (
	"errors"
	"os"
	"testing"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

func TestHookManager(t *testing.T) {
	socket := ""
	log := logger.New(os.Stderr, false)

	type mockHookData struct {
		executeCompleted  bool
		shutdownCompleted bool
	}

	hookFuncExecute := func(socket *string, log *logger.Logger, hookData any) error {
		data, ok := hookData.(*mockHookData)
		if !ok {
			return errors.New("hookData is not a *mockHookData")
		}
		data.executeCompleted = true
		return nil
	}

	hookFuncShutdown := func(socket *string, log *logger.Logger, hookData any) error {
		data, ok := hookData.(*mockHookData)
		if !ok {
			return errors.New("hookData is not a *mockHookData")
		}
		data.shutdownCompleted = true
		return nil
	}

	hm := NewHookManager()

	hookData1 := &mockHookData{
		executeCompleted:  false,
		shutdownCompleted: false,
	}
	hookData2 := &mockHookData{
		executeCompleted:  false,
		shutdownCompleted: false,
	}

	hm.Register(NewHook("TestCase1", hookFuncExecute, hookFuncShutdown, hookData1))
	hm.Register(NewHook("TestCase2", hookFuncExecute, hookFuncShutdown, hookData2))

	hm.Execute(&socket, log)

	if !hookData1.executeCompleted {
		t.Errorf("hookData1.executeCompleted = %v, want %v", hookData1.executeCompleted, true)
	}
	if !hookData2.executeCompleted {
		t.Errorf("hookData2.executeCompleted = %v, want %v", hookData2.executeCompleted, true)
	}
	if hookData1.shutdownCompleted {
		t.Errorf("hookData1.shutdownCompleted = %v, want %v", hookData1.shutdownCompleted, false)
	}
	if hookData2.shutdownCompleted {
		t.Errorf("hookData2.shutdownCompleted = %v, want %v", hookData2.shutdownCompleted, false)
	}

	hm.Shutdown(&socket, log)

	if !hookData1.shutdownCompleted {
		t.Errorf("hookData1.shutdownCompleted = %v, want %v", hookData1.shutdownCompleted, true)
	}
	if !hookData2.shutdownCompleted {
		t.Errorf("hookData2.shutdownCompleted = %v, want %v", hookData2.shutdownCompleted, true)
	}
}
