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

package stubstatus

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"

	"github.com/elastic/beats/v8/libbeat/common"
)

var (
	activeRe  = regexp.MustCompile("Active connections: (\\d+)")
	requestRe = regexp.MustCompile("\\s(\\d+)\\s+(\\d+)\\s+(\\d+)")
	connRe    = regexp.MustCompile("Reading: (\\d+) Writing: (\\d+) Waiting: (\\d+)")
)

// Map body to MapStr
func eventMapping(scanner *bufio.Scanner, m *MetricSet) (common.MapStr, error) {
	// Nginx stub status sample:
	// Active connections: 1
	// server accepts handled requests
	//  7 7 19
	// Reading: 0 Writing: 1 Waiting: 0
	var (
		active   int
		accepts  int
		handled  int
		dropped  int
		requests int
		current  int
		reading  int
		writing  int
		waiting  int
	)

	// Parse active connections.
	scanner.Scan()
	matches := activeRe.FindStringSubmatch(scanner.Text())
	if matches == nil {
		return nil, fmt.Errorf("cannot parse active connections from Nginx stub status")
	}

	active, _ = strconv.Atoi(matches[1])

	// Skip request status headers.
	scanner.Scan()

	// Parse request status.
	scanner.Scan()
	matches = requestRe.FindStringSubmatch(scanner.Text())
	if matches == nil {
		return nil, fmt.Errorf("cannot parse request status from Nginx stub status")
	}

	accepts, _ = strconv.Atoi(matches[1])
	handled, _ = strconv.Atoi(matches[2])
	requests, _ = strconv.Atoi(matches[3])

	// Derived request status.
	dropped = accepts - handled
	current = requests - m.previousNumRequests

	// Kept for next run.
	m.previousNumRequests = requests

	// Parse connection status.
	scanner.Scan()
	matches = connRe.FindStringSubmatch(scanner.Text())
	if matches == nil {
		return nil, fmt.Errorf("cannot parse connection status from Nginx stub status")
	}

	reading, _ = strconv.Atoi(matches[1])
	writing, _ = strconv.Atoi(matches[2])
	waiting, _ = strconv.Atoi(matches[3])

	event := common.MapStr{
		"hostname": m.Host(),
		"active":   active,
		"accepts":  accepts,
		"handled":  handled,
		"dropped":  dropped,
		"requests": requests,
		"current":  current,
		"reading":  reading,
		"writing":  writing,
		"waiting":  waiting,
	}

	return event, nil
}
