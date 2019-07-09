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
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"text/template"

	"github.com/elastic/beats/dev-tools/mage"
)

// moduleData provides module-level data that will be used to populate the module list
type moduleData struct {
	Path       string
	Base       string
	Title      string `yaml:"title"`
	Release    string `yaml:"release"`
	Dashboards bool
	Settings   []string `yaml:"settings"`
	CfgFile    string
	Asciidoc   string
	Metricsets []metricsetData
}

type metricsetData struct {
	Path       string
	Title      string
	Link       string
	Release    string
	DataExists bool
}

func writeTemplate(filename string, t *template.Template, args interface{}) error {
	fd, err := os.Create(filename)
	if err != nil {
		return errors.Wrapf(err, "error opening file at %s", filename)
	}
	defer fd.Close()
	err = t.Execute(fd, args)
	if err != nil {
		return errors.Wrap(err, "error executing template")
	}

	return nil
}

//a helper function used by the tempate engine to generate the base paths
// We're doing this because the mage.*Dir() functions will return an absolute path, which we can't just throw into the docs.
var funcMap = template.FuncMap{
	"basePath": func(path string) string {
		base := "module"
		if strings.Contains(path, mage.XPackBeatDir()) {
			base = "../x-pack/metricbeat/module"
		}
		return base
	},
}

// setupDirectory clears and re-creates the docs/modules directory.
func setupDirectory() error {

	docpath := mage.OSSBeatDir("docs/modules")

	err := os.RemoveAll(docpath)
	if err != nil {
		return err
	}

	return os.MkdirAll(docpath, 0744)

}

// getRelease gets the release tag, and errors out if one doesn't exist.
func getRelease(rel string) (string, error) {
	switch rel {
	case "ga", "beta", "experimental":
		return rel, nil
	case "":
		return "", fmt.Errorf("Missing a release string")
	default:
		return "", fmt.Errorf("unknown release tag %s", rel)
	}
}

// createDocsPath creates the path for the entire docs/ folder
func createDocsPath(module string) error {
	return os.MkdirAll(mage.OSSBeatDir(filepath.Join("docs/modules", module)), 0755)
}

// testIfDocsInDir tests for a `_meta/docs.asciidoc` in a given directory
func testIfDocsInDir(moduleDir string) (bool, error) {
	moduledir, err := os.Stat(moduleDir)
	if err != nil {
		return false, err
	}
	if moduledir.Mode().IsRegular() {
		return false, nil
	}
	_, err = os.Stat(filepath.Join(moduleDir, "_meta/docs.asciidoc"))
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, "error looking for asciidoc")
	}
	return true, nil
}

// loadModuleFields loads the module-specific fields.yml file
func loadModuleFields(file string) (moduleData, error) {
	fd, err := ioutil.ReadFile(file)
	if err != nil {
		return moduleData{}, errors.Wrap(err, "failed to read from spec file")
	}
	// Cheat and use the same struct.
	var mod []moduleData
	if err = yaml.Unmarshal(fd, &mod); err != nil {
		return mod[0], err
	}
	module := mod[0]

	rel, err := getRelease(module.Release)
	if err != nil {
		return mod[0], errors.Wrapf(err, "file %s is missing a release string", file)
	}
	module.Release = rel

	return module, nil
}

// getReleaseState gets the release tag in the metricset-level fields.yml, since that's all we need from that file
func getReleaseState(metricsetPath string) (string, error) {
	raw, err := ioutil.ReadFile(metricsetPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to read from spec file")
	}

	type metricset struct {
		Release string `yaml:"release"`
	}
	var rel []metricset
	if err = yaml.Unmarshal(raw, &rel); err != nil {
		return "", err
	}

	relString, err := getRelease(rel[0].Release)
	if err != nil {
		return "", errors.Wrapf(err, "metricset %s is missing a release tag", metricsetPath)
	}
	return relString, nil
}

// hasDashboards checks to see if the metricset has dashboards
func hasDashboards(modulePath string) bool {
	_, err := os.Stat(filepath.Join(modulePath, "_meta/kibana"))
	if err != nil {
		return false
	}

	return true
}

// getConfigfile uses the config.reference.yml file if it exists. if not, the normal one.
func getConfigfile(modulePath string) (string, error) {
	knownPaths := []string{"_meta/config.reference.yml", "_meta/config.yml"}
	var goodPath string
	for _, path := range knownPaths {
		testPath := filepath.Join(modulePath, path)
		_, err := os.Stat(testPath)
		if err == nil {
			goodPath = testPath
			break
		}
	}
	if goodPath == "" {
		return "", fmt.Errorf("Could not find a config file in %s", modulePath)
	}

	raw, err := ioutil.ReadFile(goodPath)
	return string(raw), err

}

// gatherMetricsets gathers all the metricsets for a given module
func gatherMetricsets(modulePath string, moduleName string) ([]metricsetData, error) {
	metricsetList, err := filepath.Glob(filepath.Join(modulePath, "/*"))
	if err != nil {
		return nil, err
	}
	var metricsets []metricsetData
	for _, metricset := range metricsetList {
		isMetricset, err := testIfDocsInDir(metricset)
		if err != nil {
			return nil, err
		}
		if !isMetricset {
			continue
		}
		metricsetName := filepath.Base(metricset)
		release, err := getReleaseState(filepath.Join(metricset, "_meta/fields.yml"))
		if err != nil {
			return nil, err
		}

		// generate the asciidoc link used in the module docs, since we need this in a few places
		link := fmt.Sprintf("<<metricbeat-metricset-%s-%s,%s>>", moduleName, metricsetName, metricsetName)

		// test to see if the metricset has a data.json
		hasData := false
		_, err = os.Stat(filepath.Join(metricset, "_meta/data.json"))
		if err == nil {
			hasData = true
		}

		ms := metricsetData{
			Path:       metricset,
			Title:      metricsetName,
			Release:    release,
			Link:       link,
			DataExists: hasData,
		}

		metricsets = append(metricsets, ms)

	} // end of metricset loop

	return metricsets, nil
}

// gatherData gathers all the data we need to construct the docs that end up in metricbeat/docs
func gatherData(modules []string) ([]moduleData, error) {
	moduleList := make([]moduleData, 0)
	//iterate over all the modules, checking to make sure we have an asciidoc file
	for _, module := range modules {

		isModule, err := testIfDocsInDir(module)
		if err != nil {
			return moduleList, err
		}
		if !isModule {
			continue
		}
		moduleName := filepath.Base(module)

		err = createDocsPath(moduleName)
		if err != nil {
			return moduleList, err
		}

		fieldsm, err := loadModuleFields(filepath.Join(module, "_meta/fields.yml"))
		if err != nil {
			return moduleList, err
		}

		cfgPath, err := getConfigfile(module)
		if err != nil {
			return moduleList, err
		}

		metricsets, err := gatherMetricsets(module, moduleName)
		if err != nil {
			return moduleList, err
		}

		//dump the contents of the module asciidoc
		moduleDoc, err := ioutil.ReadFile(filepath.Join(module, "_meta/docs.asciidoc"))
		if err != nil {
			return moduleList, err
		}

		fieldsm.Path = module
		fieldsm.CfgFile = cfgPath
		fieldsm.Metricsets = metricsets
		fieldsm.Asciidoc = string(moduleDoc)
		fieldsm.Dashboards = hasDashboards(module)
		fieldsm.Base = moduleName

		moduleList = append(moduleList, fieldsm)

	} // end of modules loop

	return moduleList, nil
}

// writeModuleDocs writes the module-level docs
func writeModuleDocs(modules []moduleData, t *template.Template) error {
	for _, mod := range modules {
		filename := mage.OSSBeatDir(filepath.Join("docs", "modules", fmt.Sprintf("%s.asciidoc", mod.Base)))
		err := writeTemplate(filename, t.Lookup("moduleDoc.tmpl"), mod)
		if err != nil {
			return err
		}
	}
	return nil
}

// writeMetricsetDocs writes the metricset-level docs
func writeMetricsetDocs(modules []moduleData, t *template.Template) error {
	for _, mod := range modules {
		for _, metricset := range mod.Metricsets {
			modData := struct {
				Mod       moduleData
				Metricset metricsetData
			}{
				mod,
				metricset,
			}
			filename := mage.OSSBeatDir(filepath.Join("docs", "modules", mod.Base, fmt.Sprintf("%s.asciidoc", metricset.Title)))
			err := writeTemplate(filename, t.Lookup("metricsetDoc.tmpl"), modData)
			if err != nil {
				return errors.Wrapf(err, "error opening file at %s", filename)
			}
		} // end metricset loop
	} // end module loop
	return nil
}

// writeModuleList writes the module linked list
func writeModuleList(modules []moduleData, t *template.Template) error {
	// Turn the map into a sorted list
	//Normally the glob functions would do this sorting for us,
	//but because we mix the regular and x-pack dirs we have to sort them again.
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Base < modules[j].Base
	})
	//write and execute the template
	filepath := mage.OSSBeatDir(filepath.Join("docs", "modules_list.asciidoc"))
	return writeTemplate(filepath, t.Lookup("moduleList.tmpl"), modules)

}

// writeDocs writes the module data to docs/
func writeDocs(modules []moduleData) error {

	tmplList := template.New("moduleList").Option("missingkey=error").Funcs(funcMap)
	tmplList, err := tmplList.ParseGlob(mage.OSSBeatDir("scripts/mage/template/*.tmpl"))
	if err != nil {
		return errors.Wrap(err, "error parsing template files")
	}

	err = writeModuleDocs(modules, tmplList)
	if err != nil {
		return errors.Wrap(err, "error writing module docs")
	}
	err = writeMetricsetDocs(modules, tmplList)
	if err != nil {
		return errors.Wrap(err, "error writing metricset docs")
	}

	err = writeModuleList(modules, tmplList)
	if err != nil {
		return errors.Wrap(err, "error writing module list")
	}

	return nil
}

// CollectDocs does the following:
// Generate the module-level docs under docs/
// Generate the module lists
// Generate the metricset-level docs
// All these are 'collected' from the asciidoc files under _meta/ in each module & metricset
func CollectDocs() error {

	//create the docs/modules dir
	err := setupDirectory()
	if err != nil {
		return err
	}
	// collect modules that have an asciidoc file
	beatsModuleGlob := filepath.Join(mage.OSSBeatDir("module"), "/*/")
	modules, err := filepath.Glob(beatsModuleGlob)
	if err != nil {
		return err
	}

	// collect additional x-pack modules
	xpackModuleGlob := filepath.Join(mage.XPackBeatDir("module"), "/*/")
	xpackModules, err := filepath.Glob(xpackModuleGlob)
	if err != nil {
		return err
	}
	modules = append(modules, xpackModules...)

	moduleMap, err := gatherData(modules)
	if err != nil {
		return err
	}

	return writeDocs(moduleMap)
}
