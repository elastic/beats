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

// getJSON returns array of string values from json string
func getJSON(b, str string) ([]string, error) {
	strArr := strings.Split(str, ".")
	bNew := []byte(b)
	newIn, err := jsonInterface(string(b[0]), strArr[0], bNew)
	if err != nil {
		return nil, fmt.Errorf("error while parsing json: %v", err)
	}
	var strArray []string
	for i, sr := range strArr {
		switch sr {
		case "#":
			newIn, err = jsonArr(strArr[i+1], newIn)
			if err != nil {
				return nil, fmt.Errorf("error while parsing json: %v", err)
			}
			for _, ir := range newIn.([]interface{}) {
				if reflect.TypeOf(ir).String() == "string" {
					strArray = append(strArray, ir.(string))
				} else {
					final, err := jsonNorm(strArr[i+2], ir)
					if err != nil {
						return nil, fmt.Errorf("error while parsing json: %v", err)
					}
					strArray = append(strArray, final.(string))
				}
			}
		default:
			newIn, err = jsonNorm(sr, newIn)
			if err != nil {
				return nil, fmt.Errorf("error while parsing json: %v", err)
			}
			if len(strArr) == i+1 {
				strArray = append(strArray, newIn.(string))
			}
		}
		if sr == "#" {
			break
		}
	}
	return strArray, nil
}

// jsonArr returns array of interface value from interface
func jsonArr(sr string, bNew interface{}) ([]interface{}, error) {
	var str []interface{}
	if bNew.([]interface{}) != nil {
		for _, i := range bNew.([]interface{}) {
			str = append(str, i.(map[string]interface{})[sr])
		}
	} else {
		return nil, fmt.Errorf("error while parsing json")
	}

	return str, nil
}

// jsonInterface returns interface from byte json
func jsonInterface(str, comStr string, bNew []byte) (interface{}, error) {
	var data map[string]interface{}
	if str == "{" && comStr != "#" {
		err := json.Unmarshal(bNew, &data)
		if err != nil {
			return nil, err
		}
	} else if str == "[" && comStr == "#" {
		var data1 []interface{}
		err := json.Unmarshal(bNew, &data1)
		if err != nil {
			return nil, fmt.Errorf("error parsing Array of json: ")
		}
		return data1, nil
	} else if str == comStr && comStr != "#" {
		err := json.Unmarshal(bNew, &data)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("error parsing json: ")
	}
	return data, nil
}

// jsonNorm returns interface value from interface
func jsonNorm(sr string, bNew interface{}) (interface{}, error) {
	if reflect.TypeOf(bNew).String() != "map[string]interface {}" {
		return nil, fmt.Errorf("error while parsing json")
	}
	return bNew.(map[string]interface{})[sr], nil
}
