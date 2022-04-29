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

package jmx

import (
	"strings"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	mbeanEventKey = "mbean"
)

type Entry struct {
	Request struct {
		Mbean     string      `json:"mbean"`
		Attribute interface{} `json:"attribute"`
	}
	Value interface{}
}

// Map responseBody to mapstr.M
//
// A response has the following structure
//  [
//    {
//        "request": {
//            "mbean": "java.lang:type=Memory",
//            "attribute": [
//                "HeapMemoryUsage",
//                "NonHeapMemoryUsage"
//            ],
//            "type": "read"
//        },
//        "value": {
//            "HeapMemoryUsage": {
//                "init": 1073741824,
//                "committed": 1037959168,
//                "max": 1037959168,
//                "used": 227420472
//            },
//            "NonHeapMemoryUsage": {
//                "init": 2555904,
//                "committed": 53477376,
//                "max": -1,
//                "used": 50519768
//            }
//        },
//        "timestamp": 1472298687,
//        "status": 200
//     }
//  ]
//
// With wildcards there is an additional nesting level:
//
//  [
//     {
//        "request": {
//           "type": "read",
//           "attribute": "maxConnections",
//           "mbean": "Catalina:name=*,type=ThreadPool"
//        },
//        "value": {
//           "Catalina:name=\"http-bio-8080\",type=ThreadPool": {
//              "maxConnections": 200
//           },
//           "Catalina:name=\"ajp-bio-8009\",type=ThreadPool": {
//              "maxConnections": 200
//           }
//        },
//        "timestamp": 1519409583
//        "status": 200,
//     }
//  ]
//
// A response with single value
//
// [
//    {
//       "request": {
//          "mbean":"java.lang:type=Runtime",
//          "attribute":"Uptime",
//          "type":"read"
//       },
//       "value":88622,
//       "timestamp":1551739190,
//       "status":200
//    }
// ]
type eventKey struct {
	mbean, event string
}

func eventMapping(entries []Entry, mapping AttributeMapping) ([]mapstr.M, error) {

	// Generate a different event for each wildcard mbean, and and additional one
	// for non-wildcard requested mbeans, group them by event name if defined
	mbeanEvents := make(map[eventKey]mapstr.M)
	var errs multierror.Errors

	for _, v := range entries {
		if v.Value == nil || v.Request.Attribute == nil {
			continue
		}

		switch attribute := v.Request.Attribute.(type) {
		case string:
			switch entryValues := v.Value.(type) {
			case float64:
				err := parseResponseEntry(v.Request.Mbean, v.Request.Mbean, attribute, entryValues, mbeanEvents, mapping)
				if err != nil {
					errs = append(errs, err)
				}
			case map[string]interface{}:
				constructEvents(entryValues, v, mbeanEvents, mapping, errs)
			}
		case []interface{}:
			entryValues := v.Value.(map[string]interface{})
			constructEvents(entryValues, v, mbeanEvents, mapping, errs)
		}
	}

	var events []mapstr.M
	for _, event := range mbeanEvents {
		events = append(events, event)
	}

	return events, errs.Err()
}

func constructEvents(entryValues map[string]interface{}, v Entry, mbeanEvents map[eventKey]mapstr.M, mapping AttributeMapping, errs multierror.Errors) {
	hasWildcard := strings.Contains(v.Request.Mbean, "*")
	for attribute, value := range entryValues {
		if !hasWildcard {
			err := parseResponseEntry(v.Request.Mbean, v.Request.Mbean, attribute, value, mbeanEvents, mapping)
			if err != nil {
				errs = append(errs, err)
			}
			continue
		}

		// If there was a wildcard, we are going to have an additional
		// nesting level in response values, and attribute here is going
		// to be actually the matching mbean name
		values, ok := value.(map[string]interface{})
		if !ok {
			errs = append(errs, errors.Errorf("expected map of values for %s", v.Request.Mbean))
			continue
		}

		responseMbean := attribute
		for attribute, value := range values {
			err := parseResponseEntry(v.Request.Mbean, responseMbean, attribute, value, mbeanEvents, mapping)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
}

func selectEvent(events map[eventKey]mapstr.M, key eventKey) mapstr.M {
	event, found := events[key]
	if !found {
		event = mapstr.M{}
		if key.mbean != "" {
			event.Put(mbeanEventKey, key.mbean)
		}
		events[key] = event
	}
	return event
}

func parseResponseEntry(
	requestMbeanName string,
	responseMbeanName string,
	attributeName string,
	attributeValue interface{},
	events map[eventKey]mapstr.M,
	mapping AttributeMapping,
) error {
	field, exists := mapping.Get(requestMbeanName, attributeName)
	if !exists {
		// This shouldn't ever happen, if it does it is probably that some of our
		// assumptions when building the request and the mapping is wrong.
		logp.Debug("jolokia.jmx", "mapping: %+v", mapping)
		return errors.Errorf("metric key '%v' for mbean '%s' not found in mapping", attributeName, requestMbeanName)
	}

	var key eventKey
	key.event = field.Event
	if responseMbeanName != requestMbeanName {
		key.mbean = responseMbeanName
	}
	event := selectEvent(events, key)

	// In case the attributeValue is a map the keys are dedotted
	data := attributeValue
	switch aValue := attributeValue.(type) {
	case map[string]interface{}:
		newData := map[string]interface{}{}
		for k, v := range aValue {
			newData[common.DeDot(k)] = v
		}
		data = newData
	case float64:
		data = aValue
	}
	_, err := event.Put(field.Field, data)
	return err
}
