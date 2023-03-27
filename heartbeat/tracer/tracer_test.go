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

package tracer

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSockTracer(t *testing.T) {
	sockName, err := uuid.NewRandom()
	require.NoError(t, err)
	sockPath := filepath.Join(os.TempDir(), sockName.String())

	listenRes := make(chan []string)
	go func() {
		listenRes <- listenTilClosed(t, sockPath)
	}()

	st, err := NewSockTracer(sockPath, time.Second)
	require.NoError(t, err)

	st.Write("start")
	st.Close()

	got := <-listenRes
	require.Equal(t, got, []string{"start"})
}

func TestSockTracerWaitFail(t *testing.T) {
	waitFor := time.Second

	started := time.Now()
	_, err := NewSockTracer(filepath.Join(os.TempDir(), "garbagenonsegarbagenooonseeense"), waitFor)
	require.Error(t, err)
	require.GreaterOrEqual(t, time.Now(), started.Add(waitFor))
}

func TestSockTracerWaitSuccess(t *testing.T) {
	waitFor := 2 * time.Second
	delay := time.Microsecond * 1500

	sockName, err := uuid.NewRandom()
	require.NoError(t, err)
	sockPath := filepath.Join(os.TempDir(), sockName.String())

	fmt.Printf("KICKIT\n")
	listenCh := make(chan net.Listener)
	time.AfterFunc(delay, func() {
		listener, err := net.Listen("unix", sockPath)
		require.NoError(t, err)
		listenCh <- listener
	})

	defer (<-listenCh).Close()

	started := time.Now()
	st, err := NewSockTracer(sockPath, waitFor)
	require.NoError(t, err)
	defer st.Close()
	elapsed := time.Now().Sub(started)
	require.GreaterOrEqual(t, elapsed, delay)
}

func listenTilClosed(t *testing.T, sockPath string) []string {
	listener, err := net.Listen("unix", sockPath)
	defer listener.Close()

	conn, err := listener.Accept()
	require.NoError(t, err)
	var received []string
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		received = append(received, scanner.Text())
	}

	return received
}
