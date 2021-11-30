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

//go:build linux && cgo && withjournald
// +build linux,cgo,withjournald

package journald

import (
	"time"

	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalfield"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func eventFromFields(
	log *logp.Logger,
	timestamp uint64,
	entryFields map[string]string,
	saveRemoteHostname bool,
) beat.Event {
	created := time.Now()
	c := journalfield.NewConverter(log, nil)
	fields := c.Convert(entryFields)
	fields.Put("event.kind", "event")

	// if entry is coming from a remote journal, add_host_metadata overwrites the source hostname, so it
	// has to be copied to a different field
	if saveRemoteHostname {
		remoteHostname, err := fields.GetValue("host.hostname")
		if err == nil {
			fields.Put("log.source.address", remoteHostname)
		}
	}

	fields.Put("event.created", created)
	receivedByJournal := time.Unix(0, int64(timestamp)*1000)

	return beat.Event{
		Timestamp: receivedByJournal,
		Fields:    fields,
	}
}
