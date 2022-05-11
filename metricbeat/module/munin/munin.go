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

package munin

import (
	"bufio"
	"io"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	unknownValue = "U"
)

var (
	// Field names must match with this expression
	// http://guide.munin-monitoring.org/en/latest/reference/plugin.html#notes-on-fieldnames
	nameRegexp = regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")
)

// Node connection
type Node struct {
	conn net.Conn

	writer io.Writer
	reader *bufio.Reader
}

// Connect with a munin node
func Connect(address string, timeout time.Duration) (*Node, error) {
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return nil, err
	}
	n := &Node{conn: conn,
		writer: conn,
		reader: bufio.NewReader(conn),
	}
	// Consume and ignore first line returned by munin, it is a comment
	// about the node
	scanner := bufio.NewScanner(n.reader)
	scanner.Scan()
	return n, scanner.Err()
}

// Close node connection releasing its resources
func (n *Node) Close() error {
	return n.conn.Close()
}

// List of plugins exposed by the node
func (n *Node) List() ([]string, error) {
	_, err := io.WriteString(n.writer, "list\n")
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(n.reader)
	scanner.Scan()
	return strings.Fields(scanner.Text()), scanner.Err()
}

// Fetch metrics from munin node
func (n *Node) Fetch(plugin string, sanitize bool) (mapstr.M, error) {
	_, err := io.WriteString(n.writer, "fetch "+plugin+"\n")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch metrics for plugin '%s'", plugin)
	}

	event := mapstr.M{}
	scanner := bufio.NewScanner(n.reader)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())

		// Munin delimits metrics with a dot
		if name == "." {
			break
		}

		name = strings.TrimSuffix(name, ".value")
		if !scanner.Scan() {
			if scanner.Err() == nil {
				return nil, errors.New("unexpected EOF when expecting value")
			}
		}
		value := scanner.Text()

		if strings.Contains(name, ".") {
			logp.Debug("munin", "ignoring field name with dot '%s'", name)
			continue
		}

		if value == unknownValue {
			logp.Debug("munin", "unknown value for '%s'", name)
			continue
		}

		if sanitize && !nameRegexp.MatchString(name) {
			logp.Debug("munin", "sanitizing name with invalid characters '%s'", name)
			name = sanitizeName(name)
		}
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			event[name] = f
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return event, nil
}

var (
	invalidCharactersRegexp = regexp.MustCompile("(^[^a-zA-Z_]|[^a-zA-Z_0-9])")
)

// Mimic munin master implementation
// https://github.com/munin-monitoring/munin/blob/20abb861/lib/Munin/Master/Node.pm#L385
func sanitizeName(name string) string {
	return invalidCharactersRegexp.ReplaceAllString(name, "_")
}
