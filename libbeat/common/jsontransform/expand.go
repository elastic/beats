package jsontransform

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/pkg/errors"
)

// ExpandFields de-dots the keys in m by expanding them in-place into a
// nested object structure, merging objects as necessary. If there are any
// conflicts (i.e. a common prefix where one field is an object and another
// is a non-object), an error will be returned.
//
// Note that ExpandFields is descructive, and in the case of an error the
// map may be left in a semi-expanded state.
func ExpandFields(m common.MapStr) error {
	for k, v := range m {
		newMap, newIsMap := getMap(v)
		if newIsMap {
			if err := ExpandFields(newMap); err != nil {
				return errors.Wrapf(err, "error expanding %q", k)
			}
		}
		if dot := strings.IndexRune(k, '.'); dot < 0 {
			continue
		}

		// Delete the dotted key.
		delete(m, k)

		// Put expands k, returning the original value if any.
		//
		// If v is a map then we will merge with an existing map if any,
		// otherwise there must not be an existing value.
		old, err := m.Put(k, v)
		if err != nil {
			// Put will return an error if we attempt to insert into a non-object value.
			return fmt.Errorf("cannot expand %q: found conflicting key", k)
		}
		if old == nil {
			continue
		}
		if !newIsMap {
			return fmt.Errorf("cannot expand %q: found existing (%T) value", k, old)
		} else {
			oldMap, oldIsMap := getMap(old)
			if !oldIsMap {
				return fmt.Errorf("cannot expand %q: found conflicting key", k)
			}
			if err := mergeObjects(newMap, oldMap); err != nil {
				return errors.Wrapf(err, "cannot expand %q", k)
			}
		}
	}
	return nil
}

// mergeObjects deep merges the elements of rhs into lhs.
//
// mergeObjects will recursively combine the entries of
// objects with the same key in each object. If there exist
// two entries with the same key in each object which
// are not both objects, then an error will result.
func mergeObjects(lhs, rhs common.MapStr) error {
	for k, rhsValue := range rhs {
		lhsValue, ok := lhs[k]
		if !ok {
			lhs[k] = rhsValue
			continue
		}
		lhsMap, ok := getMap(lhsValue)
		if !ok {
			return fmt.Errorf("cannot merge %q: found (%T) value", k, lhsValue)
		}
		rhsMap, ok := getMap(rhsValue)
		if !ok {
			return fmt.Errorf("cannot merge %q: found (%T) value", k, rhsValue)
		}
		if err := mergeObjects(lhsMap, rhsMap); err != nil {
			return errors.Wrapf(err, "cannot merge %q", k)
		}
	}
	return nil
}

func getMap(v interface{}) (map[string]interface{}, bool) {
	switch v := v.(type) {
	case map[string]interface{}:
		return v, true
	case common.MapStr:
		return v, true
	}
	return nil, false
}
