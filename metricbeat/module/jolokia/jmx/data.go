package jmx

import (
	"encoding/json"
	"strings"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
)

const (
	mbeanEventKey = "mbean"
)

type Entry struct {
	Request struct {
		Mbean string `json:"mbean"`
	}
	Value map[string]interface{}
}

// Map responseBody to common.MapStr
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
//  }
type eventKey struct {
	mbean, event string
}

func eventMapping(content []byte, mapping AttributeMapping) ([]common.MapStr, error) {
	var entries []Entry
	if err := json.Unmarshal(content, &entries); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal jolokia JSON response '%v'", string(content))
	}

	// Generate a different event for each wildcard mbean, and and additional one
	// for non-wildcard requested mbeans, group them by event name if defined
	mbeanEvents := make(map[eventKey]common.MapStr)
	var errs multierror.Errors

	for _, v := range entries {
		hasWildcard := strings.Contains(v.Request.Mbean, "*")
		for attribute, value := range v.Value {
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

	var events []common.MapStr
	for _, event := range mbeanEvents {
		events = append(events, event)
	}

	return events, errs.Err()
}

func selectEvent(events map[eventKey]common.MapStr, key eventKey) common.MapStr {
	event, found := events[key]
	if !found {
		event = common.MapStr{}
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
	events map[eventKey]common.MapStr,
	mapping AttributeMapping,
) error {
	field, exists := mapping.Get(requestMbeanName, attributeName)
	if !exists {
		return errors.Errorf("metric key '%v' not found in response (%+v)", attributeName, mapping)
	}

	var key eventKey
	key.event = field.Event
	if responseMbeanName != requestMbeanName {
		key.mbean = responseMbeanName
	}
	event := selectEvent(events, key)

	// In case the attributeValue is a map the keys are dedotted
	data := attributeValue
	c, ok := data.(map[string]interface{})
	if ok {
		newData := map[string]interface{}{}
		for k, v := range c {
			newData[common.DeDot(k)] = v
		}
		data = newData
	}
	_, err := event.Put(field.Field, data)
	return err
}
