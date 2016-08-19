package outputs

// Values is a recursive key/value store for use by output plugins and publisher
// pipeline to share context-dependent values.
type Values struct {
	parent     *Values
	key, value interface{}
}

// ValueWith creates new key/value store shadowing potentially old keys.
func ValueWith(parent *Values, key interface{}, value interface{}) *Values {
	return &Values{
		parent: parent,
		key:    key,
		value:  value,
	}
}

// Append creates new key/value store from existing store by adding a new
// key/value pair potentially shadowing an already present key/value pair.
func (v *Values) Append(key, value interface{}) *Values {
	if v.IsEmpty() {
		return ValueWith(nil, key, value)
	}
	return ValueWith(v, key, value)
}

// IsEmpty returns true if key/value store is empty.
func (v *Values) IsEmpty() bool {
	return v == nil || (v.parent == nil && v.key == nil && v.value == nil)
}

// Get retrieves a value for the given key.
func (v *Values) Get(key interface{}) (interface{}, bool) {
	if v == nil {
		return nil, false
	}
	if v.key == key {
		return v.value, true
	}
	return v.parent.Get(key)
}
