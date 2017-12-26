package jmx

import (
	"encoding/json"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
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
func eventMapping(content []byte, mapping map[string]string) (common.MapStr, error) {
	var entries []Entry
	if err := json.Unmarshal(content, &entries); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal jolokia JSON response '%v'", string(content))
	}

	event := common.MapStr{}
	var errs multierror.Errors

	for _, v := range entries {
		for attribute, value := range v.Value {
			// Extend existing event
			err := parseResponseEntry(v.Request.Mbean, attribute, value, event, mapping)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	return event, errs.Err()
}

func parseResponseEntry(
	mbeanName string,
	attributeName string,
	attibuteValue interface{},
	event common.MapStr,
	mapping map[string]string,
) error {
	// Create metric name by merging mbean and attribute fields.
	var metricName = mbeanName + "_" + attributeName

	key, exists := mapping[metricName]
	if !exists {
		return errors.Errorf("metric key '%v' not found in response", metricName)
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
