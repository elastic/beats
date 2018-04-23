package fields

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

// CollectModuleFiles looks for fields.yml files under the
// specified root directory
func CollectModuleFiles(root string) ([]*YmlFile, error) {
	modules, err := ioutil.ReadDir(root)
	if err != nil {
		return nil, err
	}

	var files []*YmlFile
	for _, m := range modules {
		files = collect(m, files, root)
	}

	return files, nil
}

func collect(module os.FileInfo, files []*YmlFile, modulesPath string) []*YmlFile {
	if !module.IsDir() {
		return files
	}

	fieldsYmlPath := filepath.Join(modulesPath, module.Name(), "_meta", "fields.yml")
	if _, err := os.Stat(fieldsYmlPath); !os.IsNotExist(err) {
		files = append(files, &YmlFile{
			Path:   fieldsYmlPath,
			Indent: 0,
		})
	}

	sets, err := ioutil.ReadDir(filepath.Join(modulesPath, module.Name()))
	for _, s := range sets {
		if !s.IsDir() {
			continue
		}

		fieldsYmlPath = filepath.Join(modulesPath, module.Name(), s.Name(), "_meta", "fields.yml")
		if _, err = os.Stat(fieldsYmlPath); !os.IsNotExist(err) {
			files = append(files, &YmlFile{
				Path:   fieldsYmlPath,
				Indent: 8,
			})
		}
	}
	return files
}
