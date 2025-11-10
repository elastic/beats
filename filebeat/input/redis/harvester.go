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

package redis

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	rd "github.com/gomodule/redigo/redis"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/filebeat/harvester"
)

// Harvester contains all redis harvester data
type Harvester struct {
	id        uuid.UUID
	done      chan struct{}
	conn      rd.Conn
	forwarder *harvester.Forwarder
	logger    *logp.Logger
}

// log contains all data related to one slowlog entry
//
//	The data is in the following format:
//	1) (integer) 13
//	2) (integer) 1309448128
//	3) (integer) 30
//	4) 1) "slowlog"
//	   2) "get"
//	   3) "100"
//	5) "100.1.1.1:12345"
//	6) "client-name"
type log struct {
	id         int64
	timestamp  int64
	duration   int
	cmd        string
	key        string
	args       []string
	clientAddr string
	clientName string
}

// NewHarvester creates a new harvester with the given connection
func NewHarvester(conn rd.Conn, logger *logp.Logger) (*Harvester, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	return &Harvester{
		id:     id,
		done:   make(chan struct{}),
		conn:   conn,
		logger: logger,
	}, nil
}

// Expected response
//
// 1) "master"
// 2) (integer) 100
// 3) 1) 1) "10.0.0.2"
//       2) "6379"
//       3) "100"
//    2) 1) "10.0.0.3"
//       2) "6379"
//       3) "100"
//
// OR
//
// 1) "slave"
// 2) "10.0.0.1"
// 3) (integer) 6379
// 4) "connected"
// 5) (integer) 100

func (h *Harvester) parseReplicationRole(reply []interface{}) (string, error) {
	role, ok := reply[0].([]byte)
	if !ok {
		return "", fmt.Errorf("unexpected type for role response: %T", reply[0])
	}
	return string(role), nil
}

// Run starts a new redis harvester
func (h *Harvester) Run() error {
	defer h.conn.Close()

	select {
	case <-h.done:
		return nil
	default:
	}
	// Writes Slowlog get, slowlog reset, and role to the buffer so they are executed together
	if err := h.conn.Send("SLOWLOG", "GET"); err != nil {
		return fmt.Errorf("error sending slowlog get: %w", err)
	}
	if err := h.conn.Send("SLOWLOG", "RESET"); err != nil {
		return fmt.Errorf("error sending slowlog reset: %w", err)
	}
	if err := h.conn.Send("ROLE"); err != nil {
		return fmt.Errorf("error sending role: %w", err)
	}

	// Flush the buffer to execute all commands and receive the replies
	h.conn.Flush()

	// Receives first reply from redis which is the one from SLOWLOG GET
	logs, err := rd.Values(h.conn.Receive())
	if err != nil {
		return fmt.Errorf("error receiving slowlog data: %w", err)
	}

	// Read reply from SLOWLOG RESET
	_, err = h.conn.Receive()
	if err != nil {
		return fmt.Errorf("error receiving reset data: %w", err)
	}

	// Read reply from ROLE
	roleReply, err := rd.Values(h.conn.Receive())
	if err != nil {
		return fmt.Errorf("error receiving replication role: %w", err)
	}
	role, err := h.parseReplicationRole(roleReply)
	if err != nil {
		return fmt.Errorf("error parsing replication role: %w", err)
	}

	for _, item := range logs {
		// Stopping here means some of the slowlog events are lost!
		select {
		case <-h.done:
			return nil
		default:
		}
		entry, err := rd.Values(item, nil)
		if err != nil {
			h.logger.Errorf("Error loading slowlog values: %s", err)
			continue
		}

		var log log
		var args []string

		// Redis < 6.0 returns 4 fields, Redis >= 6.0 returns 6 fields (adds clientAddr and clientName)
		if len(entry) >= 6 {
			_, err = rd.Scan(entry, &log.id, &log.timestamp, &log.duration, &args, &log.clientAddr, &log.clientName)
		} else {
			_, err = rd.Scan(entry, &log.id, &log.timestamp, &log.duration, &args)
		}
		if err != nil {
			h.logger.Errorf("Error scanning slowlog entry: %s", err)
			continue
		}

		// This splits up the args into cmd, key, args.
		argsLen := len(args)
		if argsLen > 0 {
			log.cmd = args[0]
		}
		if argsLen > 1 {
			log.key = args[1]
		}

		// This could contain confidential data, processors should be used to drop it if needed
		if argsLen > 2 {
			log.args = args[2:]
		}

		slowlogEntry := mapstr.M{
			"id":  log.id,
			"cmd": log.cmd,
			"key": log.key,
			"duration": mapstr.M{
				"us": log.duration,
			},
			"role": role,
		}

		// Only include client fields if they are present (Redis 6.0+)
		if log.clientAddr != "" {
			slowlogEntry["clientAddr"] = log.clientAddr
		}
		if log.clientName != "" {
			slowlogEntry["clientName"] = log.clientName
		}

		if log.args != nil {
			slowlogEntry["args"] = log.args
		}

		err = h.forwarder.Send(beat.Event{
			Timestamp: time.Unix(log.timestamp, 0).UTC(),
			Fields: mapstr.M{
				"message": strings.Join(args, " "),
				"redis": mapstr.M{
					"slowlog": slowlogEntry,
				},
				"event": mapstr.M{
					"created": time.Now(),
				},
			},
		}, h.logger)
		if err != nil {
			h.logger.Errorf("Error sending beat event: %s", err)
			continue
		}
	}
	return nil
}

// Stop stops the harvester
func (h *Harvester) Stop() {
	close(h.done)
}

// ID returns the unique harvester ID
func (h *Harvester) ID() uuid.UUID {
	return h.id
}
