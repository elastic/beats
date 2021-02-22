// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

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
	port, err := getFreePort()
	if err != nil {
		panic(err)
	}
	serviceSpec := program.Spec{
		ServicePort: port,
		Name:        "serviceable",
		Cmd:         "serviceable",
		Args:        []string{fmt.Sprintf("%d", port)},
	}

	program.Supported = append(program.Supported, configurableSpec, serviceSpec)
	program.SupportedMap["configurable"] = configurableSpec
	program.SupportedMap["serviceable"] = serviceSpec

	if err := isAvailable("configurable", "1.0"); err != nil {
		panic(err)
	}
	if err := isAvailable("serviceable", "1.0"); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func TestNotSupported(t *testing.T) {
	p := getProgram("notsupported", "1.0")

	operator := getTestOperator(t, downloadPath, installPath, p)
	err := operator.start(p, nil)
	if err == nil {
		t.Fatal("was expecting error but got none")
	}
}

func TestConfigurableRun(t *testing.T) {
	p := getProgram("configurable", "1.0")

	operator := getTestOperator(t, downloadPath, installPath, p)
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
		if item.Status != state.Healthy {
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

	// wait to finish configuring
	waitFor(t, func() error {
		items := operator.State()
		item, ok := items[p.ID()]
		if ok && item.Status == state.Configuring {
			return fmt.Errorf("process still configuring")
		}
		return nil
	})

	items := operator.State()
	item0, ok := items[p.ID()]
	if !ok || item0.Status != state.Healthy {
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

	operator := getTestOperator(t, downloadPath, installPath, p)
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
		if item.Status != state.Healthy {
			return fmt.Errorf("process never went to running")
		}
		pid = item.ProcessInfo.PID
		return nil
	})
	items := operator.State()
	item, ok := items[p.ID()]
	if !ok {
		t.Fatalf("no state for process")
	}
	assert.Equal(t, map[string]interface{}{
		"status":  float64(proto.StateObserved_HEALTHY),
		"message": "Running",
	}, item.Payload)

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
		if item.Status == state.Healthy {
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
		if item.Status != state.Healthy {
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

	operator := getTestOperator(t, downloadPath, installPath, p)
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
		if item.Status != state.Healthy {
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
		if item.Status == state.Healthy {
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
		if item.Status != state.Healthy {
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

	operator := getTestOperator(t, downloadPath, installPath, p)
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
			if item.Status != state.Healthy {
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

func TestConfigurableService(t *testing.T) {
	t.Skip("Flaky test: https://github.com/elastic/beats/issues/23607")
	p := getProgram("serviceable", "1.0")

	operator := getTestOperator(t, downloadPath, installPath, p)
	if err := operator.start(p, nil); err != nil {
		t.Fatal(err)
	}
	defer operator.stop(p) // failure catch, to ensure no sub-process stays running

	// emulating a service, so we need to start the binary here in the test
	spec := p.ProcessSpec()
	cmd := exec.Command(spec.BinaryPath, fmt.Sprintf("%d", p.ServicePort()))
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Dir = filepath.Dir(spec.BinaryPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	waitFor(t, func() error {
		items := operator.State()
		item, ok := items[p.ID()]
		if !ok {
			return fmt.Errorf("no state for process")
		}
		if item.Status != state.Healthy {
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

	// wait to finish configuring
	waitFor(t, func() error {
		items := operator.State()
		item, ok := items[p.ID()]
		if ok && item.Status == state.Configuring {
			return fmt.Errorf("process still configuring")
		}
		return nil
	})

	items := operator.State()
	item0, ok := items[p.ID()]
	if !ok || item0.Status != state.Healthy {
		t.Fatalf("Process no longer running after config %#v", items)
	}

	// stop the process
	if err := operator.stop(p); err != nil {
		t.Fatalf("Failed to stop service: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		t.Fatalf("Process failed: %v", err)
	}
}

func isAvailable(name, version string) error {
	p := getProgram(name, version)
	spec := p.ProcessSpec()
	path := spec.BinaryPath
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	if s, err := os.Stat(path); err != nil || s == nil {
		return fmt.Errorf("binary not available %s", spec.BinaryPath)
	}
	return nil
}

// getFreePort finds a free port.
func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
