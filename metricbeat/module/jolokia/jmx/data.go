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
func eventMapping(content []byte, mapping map[string]string) ([]common.MapStr, error) {
	var entries []Entry
	if err := json.Unmarshal(content, &entries); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal jolokia JSON response '%v'", string(content))
	}

	// Generate a different event for each wildcard mbean, and and additional one
	// for non-wildcard requested mbeans
	mbeanEvents := make(map[string]common.MapStr)
	var errs multierror.Errors

	for _, v := range entries {
		for attribute, value := range v.Value {
			if !strings.Contains(v.Request.Mbean, "*") {
				event, found := mbeanEvents[""]
				if !found {
					event = common.MapStr{}
					mbeanEvents[""] = event
				}
				err := parseResponseEntry(v.Request.Mbean, attribute, value, event, mapping)
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
			event, found := mbeanEvents[responseMbean]
			if !found {
				event = common.MapStr{}
				event.Put(mbeanEventKey, responseMbean)
				mbeanEvents[responseMbean] = event
			}

			for attribute, value := range values {
				err := parseResponseEntry(v.Request.Mbean, attribute, value, event, mapping)
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

func parseResponseEntry(
	requestMbeanName string,
	attributeName string,
	attibuteValue interface{},
	event common.MapStr,
	mapping map[string]string,
) error {
	// Create metric name by merging mbean and attribute fields.
	var metricName = requestMbeanName + "_" + attributeName

	key, exists := mapping[metricName]
	if !exists {
		return errors.Errorf("metric key '%v' not found in response (%+v)", metricName, mapping)
	}

	var err error

	// In case the attributeValue is a map the keys are dedotted
	c, ok := attibuteValue.(map[string]interface{})
	if ok {
		newData := map[string]interface{}{}
		for k, v := range c {
			newData[common.DeDot(k)] = v
		}
		_, err = event.Put(key, newData)
	} else {
		_, err = event.Put(key, attibuteValue)
	}

	return err
}
