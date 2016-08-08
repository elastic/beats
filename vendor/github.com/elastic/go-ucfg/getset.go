package ucfg

// ******************************************************************************
// Low level getters and setters (do we actually need this?)
// ******************************************************************************

func convertErr(opts *options, v value, err error, to string) Error {
	if err == nil {
		return nil
	}
	return raiseConversion(opts, v, err, to)
}

// number of elements for this field. If config value is a list, returns number
// of elements in list
func (c *Config) CountField(name string, opts ...Option) (int, error) {
	if name == "" {
		return len(c.fields.array()) + len(c.fields.dict()), nil
	}

	if v, ok := c.fields.get(name); ok {
		return v.Len(makeOptions(opts))
	}
	return -1, raiseMissing(c, name)
}

func (c *Config) Bool(name string, idx int, opts ...Option) (bool, error) {
	O := makeOptions(opts)
	v, err := c.getField(name, idx, O)
	if err != nil {
		return false, err
	}
	b, fail := v.toBool(O)
	return b, convertErr(O, v, fail, "bool")
}

func (c *Config) String(name string, idx int, opts ...Option) (string, error) {
	O := makeOptions(opts)
	v, err := c.getField(name, idx, O)
	if err != nil {
		return "", err
	}
	s, fail := v.toString(O)
	return s, convertErr(O, v, fail, "string")
}

func (c *Config) Int(name string, idx int, opts ...Option) (int64, error) {
	O := makeOptions(opts)
	v, err := c.getField(name, idx, O)
	if err != nil {
		return 0, err
	}
	i, fail := v.toInt(O)
	return i, convertErr(O, v, fail, "int")
}

func (c *Config) Uint(name string, idx int, opts ...Option) (uint64, error) {
	O := makeOptions(opts)
	v, err := c.getField(name, idx, O)
	if err != nil {
		return 0, err
	}
	u, fail := v.toUint(O)
	return u, convertErr(O, v, fail, "uint")
}

func (c *Config) Float(name string, idx int, opts ...Option) (float64, error) {
	O := makeOptions(opts)
	v, err := c.getField(name, idx, O)
	if err != nil {
		return 0, err
	}
	f, fail := v.toFloat(O)
	return f, convertErr(O, v, fail, "float")
}

func (c *Config) Child(name string, idx int, opts ...Option) (*Config, error) {
	O := makeOptions(opts)
	v, err := c.getField(name, idx, O)
	if err != nil {
		return nil, err
	}
	c, fail := v.toConfig(O)
	return c, convertErr(O, v, fail, "object")
}

func (c *Config) SetBool(name string, idx int, value bool, opts ...Option) error {
	return c.setField(name, idx, &cfgBool{b: value}, opts)
}

func (c *Config) SetInt(name string, idx int, value int64, opts ...Option) error {
	return c.setField(name, idx, &cfgInt{i: value}, opts)
}

func (c *Config) SetUint(name string, idx int, value uint64, opts ...Option) error {
	return c.setField(name, idx, &cfgUint{u: value}, opts)
}

func (c *Config) SetFloat(name string, idx int, value float64, opts ...Option) error {
	return c.setField(name, idx, &cfgFloat{f: value}, opts)
}

func (c *Config) SetString(name string, idx int, value string, opts ...Option) error {
	return c.setField(name, idx, &cfgString{s: value}, opts)
}

func (c *Config) SetChild(name string, idx int, value *Config, opts ...Option) error {
	return c.setField(name, idx, cfgSub{c: value}, opts)
}

func (c *Config) getField(name string, idx int, opts *options) (value, Error) {
	p := parsePathIdx(name, opts.pathSep, idx)
	v, err := p.GetValue(c, opts)
	if err != nil {
		return v, err
	}

	if v == nil {
		return nil, raiseMissing(c, p.String())
	}
	return v, nil
}

func (c *Config) setField(name string, idx int, v value, options []Option) Error {
	opts := makeOptions(options)
	p := parsePathIdx(name, opts.pathSep, idx)

	err := p.SetValue(c, opts, v)
	if err != nil {
		return err
	}

	if opts.meta != nil {
		v.setMeta(opts.meta)
	}
	return nil
}
