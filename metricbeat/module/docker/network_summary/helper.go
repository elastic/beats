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

//go:build linux
// +build linux

package network_summary

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"github.com/elastic/go-sysinfo"

	sysinfotypes "github.com/elastic/go-sysinfo/types"
)

// NoSumStats lists "stats", often config/state values, that can't be safely summed across PIDs
var NoSumStats = []string{
	"RtoAlgorithm",
	"RtoMin",
	"RtoMax",
	"MaxConn",
	"Forwarding",
	"DefaultTTL",
}

var nsRegex = regexp.MustCompile(`\d+`)

// fetchContainerNetStats gathers the PIDs associated with a container, and then uses go-sysinfo to grab the /proc/[pid]/net counters and sum them across PIDs.
func fetchContainerNetStats(client *client.Client, timeout time.Duration, container string) (*sysinfotypes.NetworkCountersInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	inspect, err := client.ContainerInspect(ctx, container)
	if err != nil {
		return nil, errors.Wrapf(err, "error fetching stats for container %s", container)
	}
	rootPID := inspect.ContainerJSONBase.State.Pid

	proc, err := sysinfo.Process(rootPID)
	procNet, ok := proc.(sysinfotypes.NetworkCounters)
	if !ok {
		return nil, errors.Wrapf(err, "cannot fetch network counters for PID %d", rootPID)
	}

	counters, err := procNet.NetworkCounters()
	if err != nil {
		return &sysinfotypes.NetworkCountersInfo{}, errors.Wrapf(err, "error fetching network counters for PID %d", rootPID)
	}

	return counters, nil

}

// fetch the network namespace associated with the PID.
func fetchNamespace(pid int) (int, error) {
	nsLink, err := os.Readlink(filepath.Join("/proc/", fmt.Sprintf("%d", pid), "/ns/net"))
	if err != nil {
		return 0, errors.Wrap(err, "error reading network namespace link")
	}
	nsidString := nsRegex.FindString(nsLink)
	// This is minor metadata, so don't consider it an error
	if nsidString == "" {
		return 0, nil
	}

	nsID, err := strconv.Atoi(nsidString)
	if err != nil {
		return 0, errors.Wrapf(err, "error converting %s to int", nsidString)
	}
	return nsID, nil
}
