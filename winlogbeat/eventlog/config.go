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

//go:build windows

package eventlog

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/joeshaw/multierror"

	conf "github.com/elastic/elastic-agent-libs/config"
)

type validator interface {
	Validate() error
}

func readConfig(c *conf.C, config interface{}) error {
	if err := c.Unpack(config); err != nil {
		return fmt.Errorf("failed unpacking config. %w", err)
	}

	if v, ok := config.(validator); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	}

	return nil
}

type config struct {
	Name          string             `config:"name"`            // Name of the event log or channel or file.
	ID            string             `config:"id"`              // Identifier for the event log.
	XMLQuery      string             `config:"xml_query"`       // Custom query XML. Must not be used with the keys from eventlog.query.
	BatchReadSize int                `config:"batch_read_size"` // Maximum number of events that Read will return.
	IncludeXML    bool               `config:"include_xml"`
	Forwarded     *bool              `config:"forwarded"`
	SimpleQuery   query              `config:",inline"`
	NoMoreEvents  NoMoreEventsAction `config:"no_more_events"` // Action to take when no more events are available - wait or stop.
	EventLanguage uint32             `config:"language"`
}

// query contains parameters used to customize the event log data that is
// queried from the log.
type query struct {
	IgnoreOlder time.Duration `config:"ignore_older"` // Ignore records older than this period of time.
	EventID     string        `config:"event_id"`     // White-list and black-list of events.
	Level       string        `config:"level"`        // Severity level.
	Provider    []string      `config:"provider"`     // Provider (source name).
}

// NoMoreEventsAction defines what action for the reader to take when
// ERROR_NO_MORE_ITEMS is returned by the Windows API.
type NoMoreEventsAction uint8

const (
	// Wait for new events.
	Wait NoMoreEventsAction = iota
	// Stop the reader.
	Stop
)

var noMoreEventsActionNames = map[NoMoreEventsAction]string{
	Wait: "wait",
	Stop: "stop",
}

// Unpack sets the action based on the string value.
func (a *NoMoreEventsAction) Unpack(v string) error {
	v = strings.ToLower(v)
	for action, name := range noMoreEventsActionNames {
		if v == name {
			*a = action
			return nil
		}
	}
	return fmt.Errorf("invalid no_more_events action: %v", v)
}

// String returns the name of the action.
func (a NoMoreEventsAction) String() string { return noMoreEventsActionNames[a] }

// Validate validates the winEventLogConfig data and returns an error describing
// any problems or nil.
func (c *config) Validate() error {
	var errs multierror.Errors

	if c.XMLQuery != "" {
		if c.ID == "" {
			errs = append(errs, fmt.Errorf("event log is missing an 'id'"))
		}

		// Check for XML syntax errors. This does not check the validity of the query itself.
		if err := xml.Unmarshal([]byte(c.XMLQuery), &struct{}{}); err != nil {
			errs = append(errs, fmt.Errorf("invalid xml_query: %w", err))
		}

		switch {
		case c.Name != "":
			errs = append(errs, fmt.Errorf("xml_query cannot be used with 'name'"))
		case c.SimpleQuery.IgnoreOlder != 0:
			errs = append(errs, fmt.Errorf("xml_query cannot be used with 'ignore_older'"))
		case c.SimpleQuery.Level != "":
			errs = append(errs, fmt.Errorf("xml_query cannot be used with 'level'"))
		case c.SimpleQuery.EventID != "":
			errs = append(errs, fmt.Errorf("xml_query cannot be used with 'event_id'"))
		case len(c.SimpleQuery.Provider) != 0:
			errs = append(errs, fmt.Errorf("xml_query cannot be used with 'provider'"))
		}
	} else if c.Name == "" {
		errs = append(errs, fmt.Errorf("event log is missing a 'name'"))
	}

	return errs.Err()
}
