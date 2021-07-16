package ecs

import "strings"

const keySeparator = "."

type Doc map[string]interface{}
type Mapping map[string]string

// Map creates the copy of the values from the doc[src] key to the doc[dst] key where the dst can be nested '.' delimited key
// Source is expected to be a simple key name, the destination could be nested child node
func (m Mapping) Map(doc Doc) Doc {
	res := make(Doc)
	for src, dst := range m {
		val, ok := doc[src]
		if !ok {
			continue
		}
		res.Set(dst, val)
	}
	return res
}

func (d Doc) Get(key string) (val interface{}, ok bool) {
	keys := getKeys(key)
	node := d

	for i := 0; i < len(keys)-1; i++ {
		if keys[i] == "" {
			return nil, false
		}
		val, ok = node[keys[i]]
		if ok {
			node, ok = val.(Doc)
			if ok {
				continue
			} else {
				break
			}
		} else {
			break
		}
	}

	if node != nil {
		val, ok = node[keys[len(keys)-1]]
	}
	return
}

func (d Doc) Set(key string, val interface{}) {
	keys := getKeys(key)
	node := d

	// Create nested keys if needed
	for i := 0; i < len(keys)-1; i++ {
		if keys[i] == "" {
			return
		}

		inode, ok := node[keys[i]]
		if ok {
			node, ok = inode.(Doc)
		} else {
			d := make(Doc)
			node[keys[i]] = d
			node = d
		}
	}

	key = keys[len(keys)-1]
	if key == "" {
		return
	}
	node[key] = val
}

func getKeys(key string) []string {
	return strings.Split(key, keySeparator)
}
