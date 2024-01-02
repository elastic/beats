// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

/*
Package zookeeper is a Metricbeat module for ZooKeeper servers.
*/
package zookeeper

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"time"
)

// RunCommand establishes a TCP connection the ZooKeeper command port that
// accepts the four-letter ZooKeeper commands and sends the given command. It
// reads all response data received on the socket and returns an io.Reader
// containing that data.
func RunCommand(command, address string, timeout time.Duration) (io.Reader, error) {
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return nil, fmt.Errorf("connection to host '%s' failed: %w", address, err)
	}
	defer conn.Close()

	// Set read and write timeout.
	err = conn.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		return nil, err
	}

	// Write four-letter command.
	_, err = conn.Write([]byte(command))
	if err != nil {
		return nil, fmt.Errorf("writing command '%s' failed: %w", command, err)
	}

	result, err := ioutil.ReadAll(conn)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}

	return bytes.NewReader(result), nil
}

// ServerID requests the server id to the Zookeper server.
func ServerID(address string, timeout time.Duration) (string, error) {
	response, err := RunCommand("conf", address, timeout)
	if err != nil {
		return "", fmt.Errorf("execution of 'conf' command failed: %w", err)
	}

	scanner := bufio.NewScanner(response)
	for scanner.Scan() {
		if line := scanner.Text(); strings.HasPrefix(line, "serverId=") {
			fields := strings.SplitN(line, "=", 2)
			if len(fields) == 2 {
				return fields[1], nil
			}
		}
	}
	return "", errors.New("no 'serverId' found in 'conf' response")
}
