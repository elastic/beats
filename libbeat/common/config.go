package common

import (
	"flag"

	"github.com/elastic/go-ucfg"
	"github.com/elastic/go-ucfg/cfgutil"
	cfgflag "github.com/elastic/go-ucfg/flag"
	"github.com/elastic/go-ucfg/yaml"
)

type Config ucfg.Config

type flagOverwrite struct {
	config *ucfg.Config
	path   string
	value  string
}

var configOpts = []ucfg.Option{
	ucfg.PathSep("."),
	ucfg.ResolveEnv,
	ucfg.VarExp,
}

func NewConfig() *Config {
	return fromConfig(ucfg.New())
}

func NewConfigFrom(from interface{}) (*Config, error) {
	c, err := ucfg.NewFrom(from, configOpts...)
	return fromConfig(c), err
}

func MergeConfigs(cfgs ...*Config) (*Config, error) {
	config := NewConfig()
	for _, c := range cfgs {
		if err := config.Merge(c); err != nil {
			return nil, err
		}
	}
	return config, nil
}

func NewConfigWithYAML(in []byte, source string) (*Config, error) {
	opts := append(
		[]ucfg.Option{
			ucfg.MetaData(ucfg.Meta{source}),
		},
		configOpts...,
	)
	c, err := yaml.NewConfig(in, opts...)
	return fromConfig(c), err
}

func NewFlagConfig(
	set *flag.FlagSet,
	def *Config,
	name string,
	usage string,
) *Config {
	opts := append(
		[]ucfg.Option{
			ucfg.MetaData(ucfg.Meta{"command line flag"}),
		},
		configOpts...,
	)

	var to *ucfg.Config
	if def != nil {
		to = def.access()
	}

	config := cfgflag.ConfigVar(set, to, name, usage, opts...)
	return fromConfig(config)
}

func NewFlagOverwrite(
	set *flag.FlagSet,
	config *Config,
	name, path, def, usage string,
) *string {
	if config == nil {
		panic("Missing configuration")
	}
	if path == "" {
		panic("empty path")
	}

	if def != "" {
		err := config.SetString(path, -1, def)
		if err != nil {
			panic(err)
		}
	}

	f := &flagOverwrite{
		config: config.access(),
		path:   path,
		value:  def,
	}

	if set == nil {
		flag.Var(f, name, usage)
	} else {
		set.Var(f, name, usage)
	}

	return &f.value
}

func LoadFile(path string) (*Config, error) {
	c, err := yaml.NewConfigWithFile(path, configOpts...)
	return fromConfig(c), err
}

func LoadFiles(paths ...string) (*Config, error) {
	merger := cfgutil.NewCollector(nil, configOpts...)
	for _, path := range paths {
		err := merger.Add(yaml.NewConfigWithFile(path, configOpts...))
		if err != nil {
			return nil, err
		}
	}
	return fromConfig(merger.Config()), nil
}

func (c *Config) Merge(from interface{}) error {
	return c.access().Merge(from, configOpts...)
}

func (c *Config) Unpack(to interface{}) error {
	return c.access().Unpack(to, configOpts...)
}

func (c *Config) Path() string {
	return c.access().Path(".")
}

func (c *Config) PathOf(field string) string {
	return c.access().PathOf(field, ".")
}

func (c *Config) HasField(name string) bool {
	return c.access().HasField(name)
}

func (c *Config) CountField(name string) (int, error) {
	return c.access().CountField(name)
}

func (c *Config) Bool(name string, idx int) (bool, error) {
	return c.access().Bool(name, idx, configOpts...)
}

func (c *Config) String(name string, idx int) (string, error) {
	return c.access().String(name, idx, configOpts...)
}

func (c *Config) Int(name string, idx int) (int64, error) {
	return c.access().Int(name, idx, configOpts...)
}

func (c *Config) Float(name string, idx int) (float64, error) {
	return c.access().Float(name, idx, configOpts...)
}

func (c *Config) Child(name string, idx int) (*Config, error) {
	sub, err := c.access().Child(name, idx, configOpts...)
	return fromConfig(sub), err
}

func (c *Config) SetBool(name string, idx int, value bool) error {
	return c.access().SetBool(name, idx, value, configOpts...)
}

func (c *Config) SetInt(name string, idx int, value int64) error {
	return c.access().SetInt(name, idx, value, configOpts...)
}

func (c *Config) SetFloat(name string, idx int, value float64) error {
	return c.access().SetFloat(name, idx, value, configOpts...)
}

func (c *Config) SetString(name string, idx int, value string) error {
	return c.access().SetString(name, idx, value, configOpts...)
}

func (c *Config) SetChild(name string, idx int, value *Config) error {
	return c.access().SetChild(name, idx, value.access(), configOpts...)
}

func (c *Config) Enabled() bool {
	testEnabled := struct {
		Enabled bool `config:"enabled"`
	}{true}

	if c == nil {
		return false
	}
	if err := c.Unpack(&testEnabled); err != nil {
		// if unpacking fails, expect 'enabled' being set to default value
		return true
	}
	return testEnabled.Enabled
}

func fromConfig(in *ucfg.Config) *Config {
	return (*Config)(in)
}

func (c *Config) access() *ucfg.Config {
	return (*ucfg.Config)(c)
}

func (c *Config) GetFields() []string {
	return c.access().GetFields()
}

func (f *flagOverwrite) String() string {
	return f.value
}

func (f *flagOverwrite) Set(v string) error {
	opts := append(
		[]ucfg.Option{
			ucfg.MetaData(ucfg.Meta{"command line flag"}),
		},
		configOpts...,
	)

	err := f.config.SetString(f.path, -1, v, opts...)
	if err != nil {
		return err
	}
	f.value = v
	return nil
}

func (f *flagOverwrite) Get() interface{} {
	return f.value
}
