// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/elastic-agent-libs/logp"
)

type messageDecoder struct {
	config  azureInputConfig
	log     *logp.Logger
	metrics *inputMetrics
}

// Decode splits the message into multiple ones based on
// the group field provided by the configuration.
//
// `messageDecoder` supports two types of messages:
//
//  1. A message with an object with a `records`
//     field containing a list of events.
//  2. A message with a single event.
//
// (1) Here is an example of a message containing an object with
// a `records` field:
//
//	{
//	  "records": [
//	    {
//	      "time": "2019-12-17T13:43:44.4946995Z",
//	      "test": "this is some message"
//	    }
//	  ]
//	}
//
// (2) Here is an example of a message with a single event:
//
//	{
//	  "time": "2019-12-17T13:43:44.4946995Z",
//	  "test": "this is some message"
//	}
//
// The Diagnostic Settings [^1] usually produces an object with a
// `records` fields (1) when exporting data to an
// event hub. This is the most common case.
//
// [^1]: the Diagnostic Settings is the Azure component used
// to export logs and metrics from an Azure service.
func (u *messageDecoder) Decode(bMessage []byte) []string {
	var mapObject map[string][]interface{}
	var records []string

	// Clean up the message for known issues [1] where Azure services produce malformed JSON documents.
	// Sanitization occurs if options are available and the message contains an invalid JSON.
	//
	// [1]: https://learn.microsoft.com/en-us/answers/questions/1001797/invalid-json-logs-produced-for-function-apps
	if len(u.config.SanitizeOptions) != 0 && !json.Valid(bMessage) {
		bMessage = sanitize(bMessage, u.config.SanitizeOptions...)
		u.metrics.sanitizedMessages.Inc()
	}

	// check if the message is a "records" object containing a list of events
	err := json.Unmarshal(bMessage, &mapObject)
	if err == nil {
		if len(mapObject[expandEventListFromField]) > 0 {
			for _, ms := range mapObject[expandEventListFromField] {
				js, err := json.Marshal(ms)
				if err == nil {
					records = append(records, string(js))
					u.metrics.receivedEvents.Inc()
				} else {
					u.log.Errorw(fmt.Sprintf("serializing message %s", ms), "error", err)
				}
			}
		}
	} else {
		u.log.Debugf("deserializing multiple messages to a `records` object returning error: %s", err)
		// in some cases the message is an array
		var arrayObject []interface{}
		err = json.Unmarshal(bMessage, &arrayObject)
		if err != nil {
			// return entire message
			u.log.Debugf("deserializing multiple messages to an array returning error: %s", err)
			u.metrics.decodeErrors.Inc()
			return []string{string(bMessage)}
		}

		for _, ms := range arrayObject {
			js, err := json.Marshal(ms)
			if err == nil {
				records = append(records, string(js))
				u.metrics.receivedEvents.Inc()
			} else {
				u.log.Errorw(fmt.Sprintf("serializing message %s", ms), "error", err)
			}
		}
	}

	return records
}
