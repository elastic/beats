package flag

import (
	"fmt"
	"path/filepath"

	"github.com/elastic/go-ucfg"
)

type FileLoader func(name string, opts ...ucfg.Option) (*ucfg.Config, error)

func NewFlagFiles(
	cfg *ucfg.Config,
	extensions map[string]FileLoader,
	opts ...ucfg.Option,
) *FlagValue {
	return newFlagValue(cfg, opts, func(path string) (*ucfg.Config, error, error) {
		ext := filepath.Ext(path)
		loader := extensions[ext]
		if loader == nil {
			loader = extensions[""]
		}
		if loader == nil {
			// TODO: better error message?
			return nil, fmt.Errorf("no loader for file '%v' found", path), nil
		}
		cfg, err := loader(path, opts...)
		return cfg, err, nil
	})
}
