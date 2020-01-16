// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package process

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	mrand "math/rand"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
)

const (
	// DefaultTimeout is timeout for starting a process, needs to be passed as a config
	DefaultTimeout = 10 * time.Second
	// MinPortNumberKey is a minimum port new process can get for newly created GRPC server
	MinPortNumberKey = "MIN_PORT_NUMBER"
	// MaxPortNumberKey is a maximum port new process can get for newly created GRPC server
	MaxPortNumberKey = "MAX_PORT_NUMBER"
	// DefaultMinPort is used when no configuration is provided
	DefaultMinPort = 10000
	// DefaultMaxPort is used when no configuration is provided
	DefaultMaxPort = 30000

	transportUnix = "unix"
	transportTCP  = "tcp"
)

var (
	// ErrProcessStartFailedTimeout is a failure of start due to timeout
	ErrProcessStartFailedTimeout = errors.New("process failed to start due to timeout")
)

// Info groups information about fresh new process
type Info struct {
	Address string
	PID     int
	Process *os.Process
}

// Creds contains information for securing a communication
type Creds struct {
	CaCert []byte
	PK     []byte
	Cert   []byte
}

// Start starts a new process
// Returns:
// - network address of child process
// - process id
// - error
func Start(logger *logger.Logger, path string, config *Config, uid, gid int, creds *Creds, arg ...string) (processInfo *Info, err error) {
	// inject env
	grpcAddress, err := getGrpcAddress(config)
	if err != nil {
		return nil, errors.New(err, "failed to acquire grpc address")
	}

	logger.Infof("address assigned to the process '%s': '%s'", path, grpcAddress)

	env := []string{
		fmt.Sprintf("SERVER_ADDRESS=%s", grpcAddress),
	}

	// create a command
	cmd := getCmd(logger, path, env, uid, gid, arg...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	// start process
	if err := cmd.Start(); err != nil {
		return nil, errors.New(err, fmt.Sprintf("failed to start '%s'", path))
	}

	// push credentials
	err = pushCredentials(stdin, creds)

	return &Info{
		PID:     cmd.Process.Pid,
		Process: cmd.Process,
		Address: grpcAddress,
	}, err
}

// Stop stops the process based on the process id
func Stop(logger *logger.Logger, pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		// Process not found (it is already killed) we treat as a success
		return nil
	}

	return proc.Kill()
}

// Attach assumes caller knows all the details about the process
// so it just tries to attach to existing PID and returns Process
// itself for awaiter
func Attach(logger *logger.Logger, pid int) (*Info, error) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		// Process not found we are unable to attach
		return nil, err
	}

	// We are attaching to an existing process,
	// address is already known to caller
	return &Info{
		PID:     proc.Pid,
		Process: proc,
	}, nil
}

func getGrpcAddress(config *Config) (string, error) {
	if config.Transport == transportUnix && runtime.GOOS != "windows" {
		getGrpcUnixAddress()
	}

	return getGrpcTCPAddress(config.MinPortNumber, config.MaxPortNumber)
}

func getGrpcUnixAddress() (string, error) {
	for i := 0; i <= 100; i++ {
		name := randSocketName()
		if fi, err := os.Stat(name); err != nil || fi == nil {
			return name, nil
		}
	}

	return "", fmt.Errorf("free unix socket not found, retry limit reached")
}

func getGrpcTCPAddress(minPort, maxPort int) (string, error) {
	if minPort == 0 {
		minPort = DefaultMinPort
	}

	if maxPort == 0 {
		maxPort = DefaultMaxPort
	}

	jitter := (maxPort - minPort) / 3
	if jitter > 0 {
		mrand.Seed(time.Now().UnixNano())
		minPort += mrand.Intn(jitter)
	}

	for port := minPort; port <= maxPort; port++ {
		desiredAddress := fmt.Sprintf("127.0.0.1:%d", port)
		listener, _ := net.Listen("tcp", desiredAddress)
		if listener != nil {
			// we found available port
			listener.Close()
			return desiredAddress, nil
		}
	}

	return "", fmt.Errorf("port not found in range %d-%d", minPort, maxPort)
}

func randSocketName() string {
	randBytes := make([]byte, 10)
	rand.Read(randBytes)
	return filepath.Join(os.TempDir(), hex.EncodeToString(randBytes)+".sock")
}

func isInt32(val int) bool {
	return val >= 0 && val <= math.MaxInt32
}

func pushCredentials(w io.Writer, c *Creds) error {
	if c == nil {
		return nil
	}

	credbytes, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	<-time.After(1500 * time.Millisecond)
	_, err = w.Write(credbytes)
	return err
}
