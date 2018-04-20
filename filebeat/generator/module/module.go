package module

import (
	"fmt"
	"path"

	"github.com/elastic/beats/filebeat/generator"
)

// Generate generates a new module.
// If module exists, error is returned.
func Generate(module, modulesPath, beatsPath string) error {
	modulePath := path.Join(modulesPath, "module", module)
	if generator.DirExists(modulePath) {
		return fmt.Errorf("module already exists: %s", module)
	}

	err := generator.CreateDirectories(modulePath, []string{path.Join("_meta", "kibana", "6")})
	if err != nil {
		return err
	}

	replace := map[string]string{"module": module}
	templatesPath := path.Join(beatsPath, "scripts", "module")
	filesToCopy := []string{
		path.Join("_meta", "fields.yml"),
		path.Join("_meta", "docs.asciidoc"),
		path.Join("_meta", "config.yml"),
		path.Join("module.yml"),
	}

	return generator.CopyTemplates(templatesPath, modulePath, filesToCopy, replace)
}
