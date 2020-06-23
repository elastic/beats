// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
)

func TestMain(m *testing.M) {
	// init supported with test cases
	configurableSpec := program.Spec{
		Name: "configurable",
		Cmd:  "configurable",
		Args: []string{},
	}

	program.Supported = append(program.Supported, configurableSpec)

	p := getProgram("configurable", "1.0")
	spec := p.Spec()
	path := spec.BinaryPath
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	if s, err := os.Stat(path); err != nil || s == nil {
		panic(fmt.Errorf("binary not available %s", spec.BinaryPath))
	}

	os.Exit(m.Run())
}

func TestNotSupported(t *testing.T) {
	p := getProgram("notsupported", "1.0")

	operator, _ := getTestOperator(t, downloadPath, installPath, p)
	err := operator.start(p, nil)
	if err == nil {
		t.Fatal("was expecting error but got none")
	}
}

func TestConfigurableRun(t *testing.T) {
	p := getProgram("configurable", "1.0")

	operator, _ := getTestOperator(t, downloadPath, installPath, p)
	if err := operator.start(p, nil); err != nil {
		t.Fatal(err)
	}
	defer operator.stop(p) // failure catch, to ensure no sub-process stays running

	waitFor(t, func() error {
		items := operator.State()
		item, ok := items[p.ID()]
		if !ok {
			return fmt.Errorf("no state for process")
		}
		if item.Status != state.Running {
			return fmt.Errorf("process never went to running")
		}
		return nil
	})

	// try to configure
	cfg := make(map[string]interface{})
	tstFilePath := filepath.Join(os.TempDir(), fmt.Sprintf("tmp%d", rand.Uint32()))
	cfg["TestFile"] = tstFilePath
	if err := operator.pushConfig(p, cfg); err != nil {
		t.Fatalf("failed to config: %v", err)
	}

	waitFor(t, func() error {
		if s, err := os.Stat(tstFilePath); err != nil || s == nil {
			return fmt.Errorf("failed to create a file using Config call %s", tstFilePath)
		}
		return nil
	})

	items := operator.State()
	item0, ok := items[p.ID()]
	if !ok || item0.Status != state.Running {
		t.Fatalf("Process no longer running after config %#v", items)
	}
	pid := item0.ProcessInfo.PID

	// stop the process
	if err := operator.stop(p); err != nil {
		t.Fatalf("Failed to stop process with PID %d: %v", pid, err)
	}

	waitFor(t, func() error {
		items := operator.State()
		_, ok := items[p.ID()]
		if ok {
			return fmt.Errorf("state for process, should be removed")
		}
		return nil
	})

	// check process stopped
	proc, err := os.FindProcess(pid)
	if err != nil && proc != nil {
		t.Fatal("Process found")
	}
}

func TestConfigurableFailed(t *testing.T) {
	p := getProgram("configurable", "1.0")

	operator, _ := getTestOperator(t, downloadPath, installPath, p)
	if err := operator.start(p, nil); err != nil {
		t.Fatal(err)
	}
	defer operator.stop(p) // failure catch, to ensure no sub-process stays running

	var pid int
	waitFor(t, func() error {
		items := operator.State()
		item, ok := items[p.ID()]
		if !ok {
			return fmt.Errorf("no state for process")
		}
		if item.Status != state.Running {
			return fmt.Errorf("process never went to running")
		}
		pid = item.ProcessInfo.PID
		return nil
	})

	// try to configure (with failed status)
	cfg := make(map[string]interface{})
	tstFilePath := filepath.Join(os.TempDir(), fmt.Sprintf("tmp%d", rand.Uint32()))
	cfg["TestFile"] = tstFilePath
	cfg["Status"] = proto.StateObserved_FAILED
	if err := operator.pushConfig(p, cfg); err != nil {
		t.Fatalf("failed to config: %v", err)
	}

	// should still create the file
	waitFor(t, func() error {
		if s, err := os.Stat(tstFilePath); err != nil || s == nil {
			return fmt.Errorf("failed to create a file using Config call %s", tstFilePath)
		}
		return nil
	})

	// wait for not running status
	waitFor(t, func() error {
		items := operator.State()
		item, ok := items[p.ID()]
		if !ok {
			return fmt.Errorf("no state for process")
		}
		if item.Status == state.Running {
			return fmt.Errorf("process never left running")
		}
		return nil
	})

	// don't send status anymore
	delete(cfg, "Status")
	if err := operator.pushConfig(p, cfg); err != nil {
		t.Fatalf("failed to config: %v", err)
	}

	// check that it restarted (has a new PID)
	waitFor(t, func() error {
		items := operator.State()
		item, ok := items[p.ID()]
		if !ok {
			return fmt.Errorf("no state for process")
		}
		if item.ProcessInfo == nil {
			return fmt.Errorf("in restart loop")
		}
		if pid == item.ProcessInfo.PID {
			return fmt.Errorf("process never restarted")
		}
		pid = item.ProcessInfo.PID
		return nil
	})

	waitFor(t, func() error {
		items := operator.State()
		item, ok := items[p.ID()]
		if !ok {
			return fmt.Errorf("no state for process")
		}
		if item.Status != state.Running {
			return fmt.Errorf("process never went to back to running")
		}
		return nil
	})

	// stop the process
	if err := operator.stop(p); err != nil {
		t.Fatalf("Failed to stop process with PID %d: %v", pid, err)
	}

	// check process stopped
	proc, err := os.FindProcess(pid)
	if err != nil && proc != nil {
		t.Fatal("Process found")
	}
}

func TestConfigurableCrash(t *testing.T) {
	p := getProgram("configurable", "1.0")

	operator, _ := getTestOperator(t, downloadPath, installPath, p)
	if err := operator.start(p, nil); err != nil {
		t.Fatal(err)
	}
	defer operator.stop(p) // failure catch, to ensure no sub-process stays running

	var pid int
	waitFor(t, func() error {
		items := operator.State()
		item, ok := items[p.ID()]
		if !ok {
			return fmt.Errorf("no state for process")
		}
		if item.Status != state.Running {
			return fmt.Errorf("process never went to running")
		}
		pid = item.ProcessInfo.PID
		return nil
	})

	// try to configure (with failed status)
	cfg := make(map[string]interface{})
	tstFilePath := filepath.Join(os.TempDir(), fmt.Sprintf("tmp%d", rand.Uint32()))
	cfg["TestFile"] = tstFilePath
	cfg["Crash"] = true
	if err := operator.pushConfig(p, cfg); err != nil {
		t.Fatalf("failed to config: %v", err)
	}

	// should still create the file
	waitFor(t, func() error {
		if s, err := os.Stat(tstFilePath); err != nil || s == nil {
			return fmt.Errorf("failed to create a file using Config call %s", tstFilePath)
		}
		return nil
	})

	// wait for not running status
	waitFor(t, func() error {
		items := operator.State()
		item, ok := items[p.ID()]
		if !ok {
			return fmt.Errorf("no state for process")
		}
		if item.Status == state.Running {
			return fmt.Errorf("process never left running")
		}
		return nil
	})

	// don't send crash anymore
	delete(cfg, "Crash")
	if err := operator.pushConfig(p, cfg); err != nil {
		t.Fatalf("failed to config: %v", err)
	}

	// check that it restarted (has a new PID)
	waitFor(t, func() error {
		items := operator.State()
		item, ok := items[p.ID()]
		if !ok {
			return fmt.Errorf("no state for process")
		}
		if item.ProcessInfo == nil {
			return fmt.Errorf("in restart loop")
		}
		if pid == item.ProcessInfo.PID {
			return fmt.Errorf("process never restarted")
		}
		pid = item.ProcessInfo.PID
		return nil
	})

	// let the process get back to ready
	waitFor(t, func() error {
		items := operator.State()
		item, ok := items[p.ID()]
		if !ok {
			return fmt.Errorf("no state for process")
		}
		if item.Status != state.Running {
			return fmt.Errorf("process never went to back to running")
		}
		return nil
	})

	// stop the process
	if err := operator.stop(p); err != nil {
		t.Fatalf("Failed to stop process with PID %d: %v", pid, err)
	}

	// check process stopped
	proc, err := os.FindProcess(pid)
	if err != nil && proc != nil {
		t.Fatal("Process found")
	}
}

func TestConfigurableStartStop(t *testing.T) {
	p := getProgram("configurable", "1.0")

	operator, _ := getTestOperator(t, downloadPath, installPath, p)
	defer operator.stop(p) // failure catch, to ensure no sub-process stays running

	// start and stop it 3 times
	for i := 0; i < 3; i++ {
		if err := operator.start(p, nil); err != nil {
			t.Fatal(err)
		}

		waitFor(t, func() error {
			items := operator.State()
			item, ok := items[p.ID()]
			if !ok {
				return fmt.Errorf("no state for process")
			}
			if item.Status != state.Running {
				return fmt.Errorf("process never went to running")
			}
			return nil
		})

		// stop the process
		if err := operator.stop(p); err != nil {
			t.Fatalf("Failed to stop process: %v", err)
		}

		waitFor(t, func() error {
			items := operator.State()
			_, ok := items[p.ID()]
			if ok {
				return fmt.Errorf("state for process, should be removed")
			}
			return nil
		})
	}
}
