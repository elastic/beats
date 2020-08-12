package cfgload

import (
	"io/ioutil"
	"path/filepath"

	"github.com/elastic/beats/v7/libbeat/common"
)

type Loader struct {
	Home              string
	StrictPermissions bool
}

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
