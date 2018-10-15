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
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
)

const (
	unknownValue = "U"
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

// List of items exposed by the node
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
func (n *Node) Fetch(items ...string) (common.MapStr, error) {
	var errs multierror.Errors
	event := common.MapStr{}

	for _, item := range items {
		_, err := io.WriteString(n.writer, "fetch "+item+"\n")
		if err != nil {
			errs = append(errs, err)
			continue
		}

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
					errs = append(errs, errors.New("unexpected EOF when expecting value"))
				}
				break
			}
			value := scanner.Text()

			key := fmt.Sprintf("%s.%s", item, name)

			if value == unknownValue {
				errs = append(errs, errors.Errorf("unknown value for %s", key))
				continue
			}
			if f, err := strconv.ParseFloat(value, 64); err == nil {
				event.Put(key, f)
				continue
			}
			event.Put(key, value)
		}

		if scanner.Err() != nil {
			errs = append(errs, scanner.Err())
		}
	}

	return event, errs.Err()
}
