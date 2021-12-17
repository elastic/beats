// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// parse returns array of string values from json string
func parse(jsonbyte, keyField string) ([]string, error) {
	keys := strings.Split(keyField, ".")
	jsonbytes := []byte(jsonbyte)
	jsoninterface, err := jsonInterface(string(jsonbyte[0]), keys[0], jsonbytes)
	if err != nil {
		return nil, fmt.Errorf("error while parsing json: %v", err)
	}
	var values []string
	for i, key := range keys {
		switch key {
		case "#":
			jsoninterface, err = jsonArr(keys[i+1], jsoninterface)
			if err != nil {
				return nil, fmt.Errorf("error while parsing json: %v", err)
			}
			// interate over []interface{}
			for _, ir := range jsoninterface.([]interface{}) {
				if reflect.TypeOf(ir).String() == "string" {
					values = append(values, ir.(string))
				} else {
					final, err := jsonNorm(keys[i+2], ir)
					if err != nil {
						return nil, fmt.Errorf("error while parsing json: %v", err)
					}
					values = append(values, final.(string))
				}
			}
		default:
			jsoninterface, err = jsonNorm(key, jsoninterface)
			if err != nil {
				return nil, fmt.Errorf("error while parsing json: %v", err)
			}
			if len(keys) == i+1 {
				values = append(values, jsoninterface.(string))
			}
		}
		if key == "#" {
			break
		}
	}
	return values, nil
}

// jsonArr returns array of interface value from interface
// input:
// key=a,
// inf=
// [
// 	{
// 		"a": "a_value_1",
// 	},
// 	{
// 		"a": "a_value_2",
// 	},
// ]
// output:
// ["a_value_1", "a_value_2"]
func jsonArr(key string, inf interface{}) ([]interface{}, error) {
	var infs []interface{}
	if inf.([]interface{}) != nil {
		for _, i := range inf.([]interface{}) {
			infs = append(infs, i.(map[string]interface{})[key])
		}
	} else {
		return nil, fmt.Errorf("error while parsing json")
	}

	return infs, nil
}

// jsonInterface returns interface from byte json
// input:
// key={
// comkey=a
// jsonbytes={"a":"a_value"}
// output:
// map[string]interface{}{
// 	"a": "a_value",
// }
func jsonInterface(key, comkey string, jsonbytes []byte) (interface{}, error) {
	var data map[string]interface{}
	if key == "{" && comkey != "#" {
		err := json.Unmarshal(jsonbytes, &data)
		if err != nil {
			return nil, err
		}
	} else if key == "[" && comkey == "#" {
		var data1 []interface{}
		err := json.Unmarshal(jsonbytes, &data1)
		if err != nil {
			return nil, fmt.Errorf("error parsing Array of json: ")
		}
		return data1, nil
	} else if key == comkey && comkey != "#" {
		err := json.Unmarshal(jsonbytes, &data)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("error parsing json: ")
	}
	return data, nil
}

// jsonNorm returns interface value from interface
// input:
// key=a
// inf=map[string]interface{}{
// 	"a": "a_value",
// }
// output:
// a_value

// input:
// key=a
// inf=map[string]interface{}{
// 	"a": map[string]interface{}{
// 		"b": "b_value",
// 	},
// }
// map[string]interface{}{
// 	"b": "b_value",
// }
func jsonNorm(key string, inf interface{}) (interface{}, error) {
	if reflect.TypeOf(inf).String() != "map[string]interface {}" {
		return nil, fmt.Errorf("error while parsing json")
	}
	return inf.(map[string]interface{})[key], nil
}
