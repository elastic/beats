package ucfg

type node map[string]interface{}

// C rebrands Config
type C Config

func newC() *C {
	return fromConfig(New())
}

func newCFrom(from interface{}) *C {
	c, err := NewFrom(from)
	if err != nil {
		panic(err)
	}
	return fromConfig(c)
}

func fromConfig(in *Config) *C {
	return (*C)(in)
}

func (c *C) asConfig() *Config {
	return (*Config)(c)
}

func (c *C) SetBool(name string, idx int, value bool) {
	c.asConfig().SetBool(name, idx, value)
}

func (c *C) SetInt(name string, idx int, value int64) {
	c.asConfig().SetInt(name, idx, value)
}

func (c *C) SetUint(name string, idx int, value uint64) {
	c.asConfig().SetUint(name, idx, value)
}

func (c *C) SetFloat(name string, idx int, value float64) {
	c.asConfig().SetFloat(name, idx, value)
}

func (c *C) SetString(name string, idx int, value string) {
	c.asConfig().SetString(name, idx, value)
}

func (c *C) SetChild(name string, idx int, value *C) {
	c.asConfig().SetChild(name, idx, (*Config)(value))
}
