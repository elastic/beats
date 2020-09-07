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

	rd "github.com/garyburd/redigo/redis"
	"github.com/gofrs/uuid"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/elastic/beats/v7/filebeat/harvester"
)

// Harvester contains all redis harvester data
type Harvester struct {
	id        uuid.UUID
	done      chan struct{}
	conn      rd.Conn
	name      string
	forwarder *harvester.Forwarder
}

type acl_log struct {
	countField      string
	count           int
	reasonField     string
	reason          string
	contextField    string
	context         string
	objectField     string
	object          string
	usernameField   string
	username        string
	agesecondsField string
	ageseconds      float64
	clientInfoField string
	clientInfo      string
}

type slowlog struct {
	id        int64
	timestamp int64
	duration  int
	cmd       string
	key       string
	args      []string
}

// NewHarvester creates a new harvester with the given connection
func NewHarvester(conn rd.Conn, name string) (*Harvester, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	return &Harvester{
		id:   id,
		done: make(chan struct{}),
		conn: conn,
		name: name,
	}, nil
}

// Run starts a new redis harvester
func (h *Harvester) Run() error {
	defer h.conn.Close()

	select {
	case <-h.done:
		return nil
	default:
	}

	// Writes Slowlog get or ACL log and reset both to the buffer so they are executed together
	if h.name == "slowlog" {
		h.conn.Send("SLOWLOG", "GET")
		h.conn.Send("SLOWLOG", "RESET")
	} else {
		h.conn.Send("ACL", "LOG")
		h.conn.Send("ACL", "LOG", "RESET")
	}

	// Flush the buffer to execute both commands and receive the reply from the commands above
	h.conn.Flush()

	// Receives first reply from redis which is the one from slowlog get or acl log
	logs, err := rd.Values(h.conn.Receive())
	if err != nil {
		return fmt.Errorf("error receiving log data: %s", err)
	}

	// Read reply from RESET
	_, err = h.conn.Receive()
	if err != nil {
		return fmt.Errorf("error receiving reset data: %s", err)
	}

	for _, item := range logs {
		// Stopping here means some of the log events are lost!
		select {
		case <-h.done:
			return nil
		default:
		}
		entry, err := rd.Values(item, nil)
		if err != nil {
			logp.Err("Error loading log values: %s", err)
			continue
		}

		var log acl_log

		var slog slowlog
		var args []string

		if h.name == "slowlog" {
			rd.Scan(entry, &slog.id, &slog.timestamp, &slog.duration, &args)

			// This splits up the args into cmd, key, args.
			argsLen := len(args)
			if argsLen > 0 {
				slog.cmd = args[0]
			}
			if argsLen > 1 {
				slog.key = args[1]
			}

			// This could contain confidential data, processors should be used to drop it if needed
			if argsLen > 2 {
				slog.args = args[2:]
			}
		} else {

			rd.Scan(entry, &log.countField, &log.count, &log.reasonField,
				&log.reason, &log.contextField, &log.context, &log.objectField,
				&log.object, &log.usernameField, &log.username, &log.agesecondsField,
				&log.ageseconds, &log.clientInfoField, &log.clientInfo)
		}

		acl_logEntry := common.MapStr{
			"count":    log.count,
			"reason":   log.reason,
			"object":   log.object,
			"username": log.username,
		}

		slowlogEntry := common.MapStr{
			"id":  slog.id,
			"cmd": slog.cmd,
			"key": slog.key,
			"duration": common.MapStr{
				"us": slog.duration,
			},
		}

		if slog.args != nil {
			slowlogEntry["args"] = slog.args
		}

		if h.name == "slowlog" {
			h.forwarder.Send(beat.Event{
				Timestamp: time.Unix(slog.timestamp, 0).UTC(),
				Fields: common.MapStr{
					"message": strings.Join(args, " "),
					"redis": common.MapStr{
						"slowlog": slowlogEntry,
					},
					"event": common.MapStr{
						"created": time.Now(),
					},
				},
			})
		} else {
			h.forwarder.Send(beat.Event{
				Timestamp: time.Now(),
				Fields: common.MapStr{
					"message": log.clientInfo,
					"redis": common.MapStr{
						"acl_log": acl_logEntry,
					},
					"event": common.MapStr{
						"created": time.Now(),
					},
				},
			})
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
