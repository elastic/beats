// Package cfgload provides support for reading configuration files from disk.
package cfgload

//go:generate godocdown -plain=false -output Readme.md

import (
	"io/ioutil"
	"path/filepath"

	"github.com/elastic/beats/v7/libbeat/common"
)

// Loader is used to configuration files.
type Loader struct {
	Home              string
	StrictPermissions bool
}

// ReadFiles reads and merges the configurations provided by the files slice.
// Load order depends on the the files are passed in. Settings in later files
// overwrite already existing settings.
func (r *Loader) ReadFiles(files []string) (*common.Config, error) {
	configFilePaths := make([]string, len(files))
	for i, path := range files {
		if !filepath.IsAbs(path) {
			path = filepath.Join(r.Home, path)
		}
		configFilePaths[i] = path
	}

	if r.StrictPermissions {
		for _, path := range configFilePaths {
			if err := common.OwnerHasExclusiveWritePerms(path); err != nil {
				return nil, err
			}
		}
	}

	config := common.NewConfig()
	for _, path := range configFilePaths {
		contents, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}

		fileConfig, err := common.NewConfigWithYAML(contents, path)
		if err != nil {
			return nil, err
		}

		if err = config.Merge(fileConfig); err != nil {
			return nil, err
		}
	}

	return config, nil
}
