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
		Objects []interface{}
	}

	err := json.Unmarshal(data, &kbResult)
	if err != nil {
		return nil, err
	}

	var objs []interface{}

	for _, obj := range kbResult.Objects {
		t, ok := obj.(map[string]interface{})["type"].(string)
		if !ok {
			return nil, fmt.Errorf("type key not found or not string")
		}
		if t != "index-pattern" {
			objs = append(objs, obj)
		}
	}

	result := common.MapStr{
		"objects": objs,
	}

	return result, nil
}
