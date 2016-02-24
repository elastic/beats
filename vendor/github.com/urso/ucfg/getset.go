package ucfg

// ******************************************************************************
// Low level getters and setters (do we actually need this?)
// ******************************************************************************

// number of elements for this field. If config value is a list, returns number
// of elements in list
func (c *Config) CountField(name string) (int, error) {
	if v, ok := c.fields[name]; ok {
		return v.Len(), nil
	}
	return -1, raise(ErrMissing)
}

func (c *Config) Bool(name string, idx int) (bool, error) {
	v, err := c.getField(name, idx)
	if err != nil {
		return false, err
	}
	return v.toBool()
}

func (c *Config) String(name string, idx int) (string, error) {
	v, err := c.getField(name, idx)
	if err != nil {
		return "", err
	}
	return v.toString()
}

func (c *Config) Int(name string, idx int) (int64, error) {
	v, err := c.getField(name, idx)
	if err != nil {
		return 0, err
	}
	return v.toInt()
}

func (c *Config) Float(name string, idx int) (float64, error) {
	v, err := c.getField(name, idx)
	if err != nil {
		return 0, err
	}
	return v.toFloat()
}

func (c *Config) Child(name string, idx int) (*Config, error) {
	v, err := c.getField(name, idx)
	if err != nil {
		return nil, err
	}
	return v.toConfig()
}

func (c *Config) SetBool(name string, idx int, value bool) {
	c.setField(name, idx, &cfgBool{b: value})
}

func (c *Config) SetInt(name string, idx int, value int64) {
	c.setField(name, idx, &cfgInt{i: value})
}

func (c *Config) SetFloat(name string, idx int, value float64) {
	c.setField(name, idx, &cfgFloat{f: value})
}

func (c *Config) SetString(name string, idx int, value string) {
	c.setField(name, idx, &cfgString{s: value})
}

func (c *Config) SetChild(name string, idx int, value *Config) {
	c.setField(name, idx, cfgSub{c: value})
}

func (c *Config) getField(name string, idx int) (value, error) {
	v, ok := c.fields[name]
	if !ok {
		return nil, raise(ErrMissing)
	}

	if idx >= v.Len() {
		return nil, raise(ErrIndexOutOfRange)
	}

	if arr, ok := v.(*cfgArray); ok {
		v = arr.arr[idx]
		if v == nil {
			return nil, raise(ErrMissing)
		}
		return arr.arr[idx], nil
	}
	return v, nil
}

func (c *Config) setField(name string, idx int, v value) {
	old, ok := c.fields[name]
	if !ok {
		if idx > 0 {
			slice := &cfgArray{arr: make([]value, idx+1)}
			slice.arr[idx] = v
			v = slice
		}
	} else if slice, ok := old.(*cfgArray); ok {
		for idx >= len(slice.arr) {
			slice.arr = append(slice.arr, nil)
		}
		slice.arr[idx] = v
		v = slice
	} else if idx > 0 {
		slice := &cfgArray{arr: make([]value, idx+1)}
		slice.arr[0] = old
		slice.arr[idx] = v
		v = slice
	}

	c.fields[name] = v
}
