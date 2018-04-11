package safemapstr

import (
	"strings"

	"github.com/elastic/beats/libbeat/common"
)

const alternativeKey = "value"

// Put This method implements a way to put dotted keys into a MapStr while
// ensuring they don't override each other. For example:
//
//  a := MapStr{}
//  safemapstr.Put(a, "com.docker.swarm.task", "x")
//  safemapstr.Put(a, "com.docker.swarm.task.id", 1)
//  safemapstr.Put(a, "com.docker.swarm.task.name", "foobar")
//
// Will result in `{"com":{"docker":{"swarm":{"task":{"id":1,"name":"foobar","value":"x"}}}}}`
//
// Put detects this scenario and renames the common base key, by appending
// `.value`
func Put(data common.MapStr, key string, value interface{}) error {
	// XXX This implementation mimics `common.MapStr.Put`, both should be updated to have similar behavior
	keyParts := strings.SplitN(key, ".", 2)

	// If leaf node or key exists directly
	if len(keyParts) == 1 {
		oldValue, exists := data[key]
		if exists {
			oldMap, ok := tryToMapStr(oldValue)
			if ok {
				// This would replace a whole object, change its key to avoid that:
				oldMap[alternativeKey] = value
				return nil
			}
		}
		data[key] = value
		return nil
	}

	// Checks if first part of the key exists
	k := keyParts[0]
	d, exists := data[k]
	if !exists {
		d = common.MapStr{}
		data[k] = d
	}

	v, ok := tryToMapStr(d)
	if !ok {
		// This would replace a leaf with an object, change its key to avoid that:
		v = common.MapStr{alternativeKey: d}
		data[k] = v
	}

	return Put(v, keyParts[1], value)
}

func tryToMapStr(v interface{}) (common.MapStr, bool) {
	switch m := v.(type) {
	case common.MapStr:
		return m, true
	case map[string]interface{}:
		return common.MapStr(m), true
	default:
		return nil, false
	}
}
