// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin || synthetics

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// NodeProcess represents a managed Node.js process
type NodeProcess struct {
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	stdinMutex sync.Mutex
	ready      bool
	ipcMode    string
}

// IPCMessage represents a message structure for IPC communication
type IPCMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// StartNodeProcess starts a Node.js process with IPC enabled
func StartNodeProcess(nodePath, scriptPath string, ipcMode string) (*NodeProcess, error) {
	var cmd *exec.Cmd
	var stdin io.WriteCloser
	var stdout io.ReadCloser
	var err error

	if ipcMode == "stdin" {
		// Use stdin/stdout for communication
		cmd = exec.Command(nodePath, scriptPath, "ipc", "--ipc-mode", "stdin")

		stdin, err = cmd.StdinPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
		}

		stdout, err = cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
		}

		// Redirect stderr to the current process's stderr
		cmd.Stderr = os.Stderr
	} else {
		// Use Node.js IPC mechanism
		cmd = exec.Command(nodePath, scriptPath, "ipc")
		cmd.Env = append(os.Environ(), "NODE_CHANNEL_FD=3")

		// We'll set up file descriptors for IPC
		// This is advanced and platform-specific, so simplified here
		// In a real implementation, you'd need to set up proper FD handling
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start Node.js process: %w", err)
	}

	process := &NodeProcess{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		ipcMode: ipcMode,
	}

	if ipcMode == "stdin" {
		// Start a goroutine to read messages from the process
		go process.readMessages()
	}

	return process, nil
}

// readMessages reads JSON messages from the Node.js process
func (p *NodeProcess) readMessages() {
	decoder := json.NewDecoder(p.stdout)
	for {
		var message map[string]interface{}
		err := decoder.Decode(&message)
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "Error reading from Node.js process: %v\n", err)
			}
			break
		}

		// Handle ready message
		if messageType, ok := message["type"].(string); ok && messageType == "ready" {
			p.ready = true
		}

		// Process other messages - in a real implementation you'd use channels
		// to communicate these messages back to the caller
		fmt.Printf("Received message: %v\n", message)
	}
}

// SendMessage sends a message to the Node.js process
func (p *NodeProcess) SendMessage(messageType string, payload interface{}) error {
	message := IPCMessage{
		Type:    messageType,
		Payload: payload,
	}

	p.stdinMutex.Lock()
	defer p.stdinMutex.Unlock()

	encoder := json.NewEncoder(p.stdin)
	return encoder.Encode(message)
}

// InitMainProcess initializes the main Node.js process
func (p *NodeProcess) InitMainProcess(options map[string]interface{}) error {
	return p.SendMessage("init", options)
}

// RunTests runs tests in the main process
func (p *NodeProcess) RunTests(options map[string]interface{}) error {
	return p.SendMessage("run", options)
}

// CreateWorker creates a new worker thread
func (p *NodeProcess) CreateWorker(workerId string, workerData map[string]interface{}) error {
	data := map[string]interface{}{
		"createWorker": true,
		"workerData":   workerData,
	}

	payload := map[string]interface{}{
		"workerId": workerId,
		"data":     data,
	}

	return p.SendMessage("worker", payload)
}

// SendToWorker sends data to a specific worker
func (p *NodeProcess) SendToWorker(workerId string, data map[string]interface{}) error {
	payload := map[string]interface{}{
		"workerId": workerId,
		"data":     data,
	}

	return p.SendMessage("worker", payload)
}

// StopWorker terminates a specific worker
func (p *NodeProcess) StopWorker(workerId string) error {
	payload := map[string]interface{}{
		"workerId": workerId,
	}

	return p.SendMessage("stop", payload)
}

// StopAllWorkers terminates all worker threads
func (p *NodeProcess) StopAllWorkers() error {
	return p.SendMessage("stop", map[string]interface{}{})
}

// Close terminates the Node.js process
func (p *NodeProcess) Close() error {
	// Try to gracefully terminate all workers first
	_ = p.StopAllWorkers()

	// Give the process some time to clean up
	// In a real implementation, you'd use proper synchronization

	// Kill the process
	return p.cmd.Process.Kill()
}
