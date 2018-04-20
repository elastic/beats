package fileset

import (
	"fmt"
	"path"

	"github.com/elastic/beats/filebeat/generator"
)

// Generate generates a new fileset under a module.
// If fileset exists or the module does not exists, an error is returned.
func Generate(module, fileset, modulesPath, beatsPath string) error {
	filesetPath := path.Join(modulesPath, "module", module, fileset)
	if generator.DirExists(filesetPath) {
		return fmt.Errorf("fileset already exists: %s", fileset)
	}

	err := generator.CreateDirectories(filesetPath, []string{"", "_meta", "test", "config", "ingest"})
	if err != nil {
		return err
	}

	replace := map[string]string{
		"module":  module,
		"fileset": fileset,
	}
	templatesPath := path.Join(beatsPath, "scripts", "fileset")
	filesToCopy := []string{
		path.Join("config", "config.yml"),
		path.Join("ingest", "pipeline.json"),
		"manifest.yml",
	}
	err = generator.CopyTemplates(templatesPath, filesetPath, filesToCopy, replace)
	if err != nil {
		return err
	}
	err = generator.RenameConfigYml(modulesPath, module, fileset)
	if err != nil {
		return err
	}

	return addFilesetDashboard(module, fileset, modulesPath, templatesPath)
}

func addFilesetDashboard(module, fileset, modulesPath, templatesPath string) error {
	template := path.Join(templatesPath, "module-fileset.yml")
	dest := path.Join(modulesPath, "module", module, "module.yml")
	replacement := map[string]string{"module": module, "fileset": fileset}
	return generator.AppendTemplate(template, dest, replacement)
}
