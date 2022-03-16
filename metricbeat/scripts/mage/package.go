// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package mage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/pkg/errors"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
)

const (
	dirModulesGenerated  = "build/package/module"
	dirModulesDGenerated = "build/package/modules.d"
)

// CustomizePackaging modifies the package specs to add the modules.d directory.
// And for Windows it comments out the system/load metricset because it's
// not supported. You must declare a dependency on either
// PrepareModulePackagingOSS or PrepareModulePackagingXPack.
func CustomizePackaging() {
	mg.Deps(customizeLightModulesPackaging)

	var (
		modulesDTarget = "modules.d"
		modulesD       = devtools.PackageFile{
			Mode:    0644,
			Source:  dirModulesDGenerated,
			Config:  true,
			Modules: true,
		}
		windowsModulesD = devtools.PackageFile{
			Mode:    0644,
			Source:  "{{.PackageDir}}/modules.d",
			Config:  true,
			Modules: true,
			Dep: func(spec devtools.PackageSpec) error {
				if err := devtools.Copy(dirModulesDGenerated, spec.MustExpand("{{.PackageDir}}/modules.d")); err != nil {
					return errors.Wrap(err, "failed to copy modules.d dir")
				}

				return devtools.FindReplace(
					spec.MustExpand("{{.PackageDir}}/modules.d/system.yml"),
					regexp.MustCompile(`- load`), `#- load`)
			},
		}
		windowsReferenceConfig = devtools.PackageFile{
			Mode:   0644,
			Source: "{{.PackageDir}}/metricbeat.reference.yml",
			Dep: func(spec devtools.PackageSpec) error {
				err := devtools.Copy("metricbeat.reference.yml",
					spec.MustExpand("{{.PackageDir}}/metricbeat.reference.yml"))
				if err != nil {
					return errors.Wrap(err, "failed to copy reference config")
				}

				return devtools.FindReplace(
					spec.MustExpand("{{.PackageDir}}/metricbeat.reference.yml"),
					regexp.MustCompile(`- load`), `#- load`)
			},
		}
	)

	for _, args := range devtools.Packages {
		switch args.OS {
		case "windows":
			args.Spec.Files[modulesDTarget] = windowsModulesD
			args.Spec.ReplaceFile("{{.BeatName}}.reference.yml", windowsReferenceConfig)
		default:
			pkgType := args.Types[0]
			switch pkgType {
			case devtools.TarGz, devtools.Zip, devtools.Docker:
				args.Spec.Files[modulesDTarget] = modulesD
			case devtools.Deb, devtools.RPM:
				args.Spec.Files["/etc/{{.BeatName}}/"+modulesDTarget] = modulesD
			default:
				panic(errors.Errorf("unhandled package type: %v", pkgType))
			}
		}
	}
}

// PrepareModulePackagingOSS generates build/package/modules and
// build/package/modules.d directories for use in packaging.
func PrepareModulePackagingOSS() error {
	err := prepareLightModulesPackaging("module")
	if err != nil {
		return err
	}
	return prepareModulePackaging([]struct{ Src, Dst string }{
		{devtools.OSSBeatDir("modules.d"), dirModulesDGenerated},
	}...)
}

// PrepareModulePackagingXPack generates build/package/modules and
// build/package/modules.d directories for use in packaging.
func PrepareModulePackagingXPack() error {
	err := prepareLightModulesPackaging("module", devtools.OSSBeatDir("module"))
	if err != nil {
		return err
	}
	return prepareModulePackaging([]struct{ Src, Dst string }{
		{devtools.OSSBeatDir("modules.d"), dirModulesDGenerated},
		{"modules.d", dirModulesDGenerated},
	}...)
}

func prepareModulePackaging(files ...struct{ Src, Dst string }) error {
	mg.Deps(GenerateDirModulesD)

	err := devtools.Clean([]string{
		dirModulesDGenerated,
	})
	if err != nil {
		return err
	}

	for _, copyAction := range files {
		err := (&devtools.CopyTask{
			Source:  copyAction.Src,
			Dest:    copyAction.Dst,
			Mode:    0644,
			DirMode: 0755,
		}).Execute()
		if err != nil {
			return err
		}
	}
	return nil
}

const modulesDHeader = "# Module: {{ .Module }}\n# Docs: https://www.elastic.co/guide/en/beats/{{ .BeatName }}/{{ .BeatDocBranch }}/{{ .BeatName }}-module-{{ .Module }}.html"

// GenerateDirModulesD generates the modules.d directory
func GenerateDirModulesD() error {
	if err := os.RemoveAll("modules.d"); err != nil {
		return err
	}

	shortConfigs, err := filepath.Glob("module/*/_meta/config.yml")
	if err != nil {
		return err
	}
	flavorConfigs, err := filepath.Glob("module/*/_meta/config-*.yml")
	if err != nil {
		return err
	}
	shortConfigs = append(shortConfigs, flavorConfigs...)

	docBranch, err := devtools.BeatDocBranch()
	if err != nil {
		errors.Wrap(err, "failed to get doc branch")
	}

	mode := 0644
	for _, f := range shortConfigs {
		moduleName, configName, ok := moduleConfigParts(f)
		if !ok {
			continue
		}

		suffix := ".yml.disabled"
		if configName == "system" {
			suffix = ".yml"
		}
		path := filepath.Join("modules.d", configName+suffix)

		headerArgs := map[string]interface{}{
			"Module":        moduleName,
			"BeatName":      devtools.BeatName,
			"BeatDocBranch": docBranch,
		}
		header := devtools.MustExpand(modulesDHeader, headerArgs)

		err := copyWithHeader(header, f, path, os.FileMode(mode))
		if err != nil {
			return err
		}
	}
	return nil
}

// customizeLightModulesPackaging customizes packaging to add light modules
func customizeLightModulesPackaging() error {
	var (
		moduleTarget = "module"
		module       = devtools.PackageFile{
			Mode:   0644,
			Source: dirModulesGenerated,
		}
	)

	for _, args := range devtools.Packages {
		pkgType := args.Types[0]
		switch pkgType {
		case devtools.TarGz, devtools.Zip, devtools.Docker:
			args.Spec.Files[moduleTarget] = module
		case devtools.Deb, devtools.RPM:
			args.Spec.Files["/usr/share/{{.BeatName}}/"+moduleTarget] = module
		default:
			return fmt.Errorf("unhandled package type: %v", pkgType)
		}
	}
	return nil
}

// prepareLightModulesPackaging generates light modules
func prepareLightModulesPackaging(paths ...string) error {
	err := devtools.Clean([]string{dirModulesGenerated})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dirModulesGenerated, 0755); err != nil {
		return err
	}

	filePatterns := []string{
		"*/module.yml",
		"*/*/manifest.yml",
	}

	var tasks []devtools.CopyTask
	for _, path := range paths {
		for _, pattern := range filePatterns {
			matches, err := filepath.Glob(filepath.Join(path, pattern))
			if err != nil {
				return err
			}

			for _, file := range matches {
				rel, _ := filepath.Rel(path, file)
				dest := filepath.Join(dirModulesGenerated, rel)
				tasks = append(tasks, devtools.CopyTask{
					Source:  file,
					Dest:    dest,
					Mode:    0644,
					DirMode: 0755,
				})
			}
		}
	}

	if len(tasks) == 0 {
		return fmt.Errorf("no light modules found")
	}

	for _, task := range tasks {
		if err := task.Execute(); err != nil {
			return err
		}
	}
	return nil
}

// moduleConfigParts obtain the moduleName and the configName from a config path.
// The configName includes the flavor
func moduleConfigParts(f string) (moduleName string, configName string, ok bool) {
	parts := strings.Split(filepath.ToSlash(f), "/")
	if len(parts) < 4 {
		return
	}
	moduleName = parts[1]
	configName = moduleName
	ok = true

	fileName := strings.TrimSuffix(parts[3], ".yml")
	parts = strings.SplitN(fileName, "-", 2)
	if len(parts) > 1 {
		configName = moduleName + "-" + parts[1] // module + flavor
	}
	return
}

// copyWithHeader copies a file from `src` to `dst` adding a `header` in the destination file
func copyWithHeader(header, src, dst string, mode os.FileMode) error {
	dstFile, err := os.OpenFile(devtools.CreateDir(dst), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode&os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "failed to open copy destination")
	}
	defer dstFile.Close()

	_, err = io.WriteString(dstFile, header+"\n\n")
	if err != nil {
		return errors.Wrap(err, "failed to write header")
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return errors.Wrap(err, "failed to open copy source")
	}
	defer srcFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return errors.Wrap(err, "failed to copy file")
}
