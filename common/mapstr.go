package common

// Commonly used map of things, used in JSON creation and the like.
type MapStr map[string]interface{}

// MapStrUnion creates a new MapStr containing the union of the
// key-value pairs of the two maps. If the same key is present in
// both, the key-value pairs from dict2 overwrite the ones from dict1.
func MapStrUnion(dict1 MapStr, dict2 MapStr) MapStr {
	dict := MapStr{}

	for k, v := range dict1 {
		dict[k] = v
	}

	for k, v := range dict2 {
		dict[k] = v
	}
	return dict
}

// Update copies all the key-value pairs from the
// d map overwriting any existing keys.
func (m MapStr) Update(d MapStr) {
	for k, v := range d {
		m[k] = v
	}
}
