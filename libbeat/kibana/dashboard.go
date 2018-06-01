package kibana

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
)

// RemoveIndexPattern removes the index pattern entry from a given dashboard export
func RemoveIndexPattern(data []byte) (common.MapStr, error) {

	var kbResult struct {
		// Has to be defined as interface instead of Type directly as it has to be assigned again
		// and otherwise would not contain the full content.
		Objects []common.MapStr
	}

	var result common.MapStr
	// Full struct need to not loose any data
	err := json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	// For easier handling, unmarshal into predefined struct
	err = json.Unmarshal(data, &kbResult)
	if err != nil {
		return nil, err
	}

	var objs []interface{}

	for _, obj := range kbResult.Objects {
		v, err := obj.GetValue("type")
		if err != nil {
			return nil, fmt.Errorf("type key not found or not string")
		}
		if v != "index-pattern" {
			objs = append(objs, obj)
		}
	}

	result["objects"] = objs

	return result, nil
}
