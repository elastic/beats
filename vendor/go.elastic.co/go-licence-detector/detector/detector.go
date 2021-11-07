// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:generate pkger -include=go.elastic.co/go-licence-detector:/assets -o=detector

package detector // import "go.elastic.co/go-licence-detector/detector"

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/google/licenseclassifier"
	"github.com/karrick/godirwalk"
	"github.com/markbates/pkger"
	"go.elastic.co/go-licence-detector/dependency"
)

const (
	// detectionThreshold is the minimum confidence score required from the licence classifier.
	detectionThreshold = 0.85
	licenceDBPath      = "go.elastic.co/go-licence-detector:/assets/licence.db"
)

var errLicenceNotFound = errors.New("failed to detect licence")

type dependencies struct {
	direct   []*module
	indirect []*module
}

type module struct {
	Path     string     // module path
	Version  string     // module version
	Main     bool       // is this the main module?
	Time     *time.Time // time version was created
	Indirect bool       // is this module only an indirect dependency of main module?
	Dir      string     // directory holding files for this module, if any
	Replace  *module    // replace directive
}

// NewClassifier creates a new instance of the licence classifier.
func NewClassifier(dataPath string) (*licenseclassifier.License, error) {
	if dataPath == "" {
		return newClassiferFromEmbeddedDB()
	}

	absPath, err := filepath.Abs(dataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to determine absolute path of licence data file: %w", err)
	}

	return licenseclassifier.New(detectionThreshold, licenseclassifier.Archive(absPath))
}

func newClassiferFromEmbeddedDB() (*licenseclassifier.License, error) {
	f, err := pkger.Open(licenceDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open bundled licence database: %w", err)
	}

	defer f.Close()

	dbBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read bundled licence database: %w", err)
	}

	return licenseclassifier.New(detectionThreshold, licenseclassifier.ArchiveBytes(dbBytes))
}

// Detect searches the dependencies on disk and detects licences.
func Detect(data io.Reader, classifier *licenseclassifier.License, rules *Rules, overrides dependency.Overrides, includeIndirect bool) (*dependency.List, error) {
	// parse the output of go mod list
	deps, err := parseDependencies(data, includeIndirect)
	if err != nil {
		return nil, err
	}

	// find licences for each dependency
	return detectLicences(classifier, rules, deps, overrides)
}

func parseDependencies(data io.Reader, includeIndirect bool) (*dependencies, error) {
	deps := &dependencies{}
	decoder := json.NewDecoder(data)
	for {
		var mod module
		if err := decoder.Decode(&mod); err != nil {
			if errors.Is(err, io.EOF) {
				return deps, nil
			}
			return deps, fmt.Errorf("failed to parse dependencies: %w", err)
		}

		if !mod.Main && mod.Dir != "" {
			if mod.Indirect {
				if includeIndirect {
					deps.indirect = append(deps.indirect, &mod)
				}
				continue
			}
			deps.direct = append(deps.direct, &mod)
		}
	}
}

func detectLicences(classifier *licenseclassifier.License, rules *Rules, deps *dependencies, overrides dependency.Overrides) (*dependency.List, error) {
	depList := &dependency.List{}
	licenceRegex := buildLicenceRegex()

	var err error
	if depList.Direct, err = doDetectLicences(licenceRegex, classifier, rules, deps.direct, overrides); err != nil {
		return depList, err
	}

	if depList.Indirect, err = doDetectLicences(licenceRegex, classifier, rules, deps.indirect, overrides); err != nil {
		return depList, err
	}

	return depList, nil
}

func doDetectLicences(licenceRegex *regexp.Regexp, classifier *licenseclassifier.License, rules *Rules, depList []*module, overrides dependency.Overrides) ([]dependency.Info, error) {
	if len(depList) == 0 {
		return nil, nil
	}

	depInfoList := make([]dependency.Info, len(depList))
	for i, mod := range depList {
		depInfo := mkDepInfo(mod, overrides)

		// find the licence file if the override hasn't provided one
		if depInfo.LicenceFile == "" {
			var err error
			depInfo.LicenceFile, err = findLicenceFile(depInfo.Dir, licenceRegex)
			if err != nil && !errors.Is(err, errLicenceNotFound) {
				return nil, fmt.Errorf("failed to find licence file for %s in %s: %w", depInfo.Name, depInfo.Dir, err)
			}
		} else if depInfo.LicenceTextOverrideFile == "" {
			// if licence file is given but no overrides, use the selected licence file
			licFile, err := securejoin.SecureJoin(depInfo.Dir, depInfo.LicenceFile)
			if err != nil {
				return nil, fmt.Errorf("failed to generate secure path to licence file of %s: %w", depInfo.Name, err)
			}
			depInfo.LicenceFile = licFile
		}

		// detect the licence type if the override hasn't provided one
		if depInfo.LicenceType == "" {
			if depInfo.LicenceFile == "" {
				return nil, fmt.Errorf("no licence file found for %s. Add an override entry with licence type to continue.", depInfo.Name)
			}

			var err error
			depInfo.LicenceType, err = detectLicenceType(classifier, depInfo.LicenceFile)
			if err != nil {
				return nil, fmt.Errorf("failed to detect licence type of %s from %s: %w", depInfo.Name, depInfo.LicenceFile, err)
			}

			if depInfo.LicenceType == "" {
				return nil, fmt.Errorf("licence unknown for %s. Add an override entry with licence type to continue.", depInfo.Name)
			}
		}

		if !rules.IsAllowed(depInfo.LicenceType) {
			return nil, fmt.Errorf("dependency %s uses licence %s which is not allowed by the rules file", depInfo.Name, depInfo.LicenceType)
		}

		depInfoList[i] = depInfo
	}

	return depInfoList, nil
}

func mkDepInfo(mod *module, overrides dependency.Overrides) dependency.Info {
	m := mod
	if mod.Replace != nil {
		m = mod.Replace
	}

	override, ok := overrides[m.Path]
	if !ok {
		override = dependency.Info{}
	}

	return dependency.Info{
		Name:                    m.Path,
		Dir:                     coalesce(override.Dir, m.Dir),
		Version:                 coalesce(override.Version, m.Version),
		VersionTime:             coalesce(override.VersionTime, m.Time.Format(time.RFC3339)),
		URL:                     determineURL(override.URL, m.Path),
		LicenceFile:             override.LicenceFile,
		LicenceType:             override.LicenceType,
		LicenceTextOverrideFile: override.LicenceTextOverrideFile,
	}
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}

	return b
}

func determineURL(overrideURL, modulePath string) string {
	if overrideURL != "" {
		return overrideURL
	}

	parts := strings.Split(modulePath, "/")
	switch parts[0] {
	case "github.com":
		// GitHub URLs that have more than two path elements will return a 404 (e.g. https://github.com/elazarl/goproxy/ext).
		// We strip out the extra path elements from the end to come up with a valid URL like https://github.com/elazarl/goproxy/.
		if len(parts) > 3 {
			return "https://" + strings.Join(parts[:3], "/")
		}
		return "https://" + modulePath
	case "k8s.io":
		return "https://github.com/kubernetes/" + parts[1]
	default:
		return "https://" + modulePath
	}
}

func buildLicenceRegex() *regexp.Regexp {
	// inspired by https://github.com/src-d/go-license-detector/blob/7961dd6009019bc12778175ef7f074ede24bd128/licensedb/internal/investigation.go#L29
	licenceFileNames := []string{
		`li[cs]en[cs]es?`,
		`legal`,
		`copy(left|right|ing)`,
		`unlicense`,
		`l?gpl([-_ v]?)(\d\.?\d)?`,
		`bsd`,
		`mit`,
		`apache`,
	}

	regexStr := fmt.Sprintf(`^(?i:(%s)(\.(txt|md|rst))?)$`, strings.Join(licenceFileNames, "|"))
	return regexp.MustCompile(regexStr)
}

func findLicenceFile(root string, licenceRegex *regexp.Regexp) (string, error) {
	errStopWalk := errors.New("stop walk")
	var licenceFile string
	err := godirwalk.Walk(root, &godirwalk.Options{
		Callback: func(osPathName string, dirent *godirwalk.Dirent) error {
			if licenceRegex.MatchString(dirent.Name()) {
				if dirent.IsDir() {
					return filepath.SkipDir
				}
				licenceFile = osPathName
				return errStopWalk
			}
			return nil
		},
		Unsorted: false,
	})

	if err != nil {
		if errors.Is(err, errStopWalk) {
			return licenceFile, nil
		}
		return "", err
	}

	return "", errLicenceNotFound
}

func detectLicenceType(classifier *licenseclassifier.License, licenceFile string) (string, error) {
	contents, err := ioutil.ReadFile(licenceFile)
	if err != nil {
		return "", fmt.Errorf("failed to read licence content from %s: %w", licenceFile, err)
	}

	matches := classifier.MultipleMatch(string(contents), true)
	// there should be at least one match
	if len(matches) < 1 {
		return "", fmt.Errorf("failed to detect licence type of %s", licenceFile)
	}

	// matches are sorted by confidence such that the first result has the highest confidence level
	return matches[0].Name, nil
}
