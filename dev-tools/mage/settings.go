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
	"go/build"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/vcs"
)

const (
	fpmVersion = "1.11.0"

	// Docker images. See https://github.com/elastic/golang-crossbuild.
	beatsFPMImage = "docker.elastic.co/beats-dev/fpm"
	// BeatsCrossBuildImage is the image used for crossbuilding Beats.
	BeatsCrossBuildImage = "docker.elastic.co/beats-dev/golang-crossbuild"

	elasticBeatsImportPath = "github.com/elastic/beats"
)

// Common settings with defaults derived from files, CWD, and environment.
var (
	GOOS         = build.Default.GOOS
	GOARCH       = build.Default.GOARCH
	GOARM        = EnvOr("GOARM", "")
	Platform     = MakePlatformAttributes(GOOS, GOARCH, GOARM)
	BinaryExt    = ""
	XPackDir     = "../x-pack"
	RaceDetector = false
	TestCoverage = false

	BeatName        = EnvOr("BEAT_NAME", filepath.Base(CWD()))
	BeatServiceName = EnvOr("BEAT_SERVICE_NAME", BeatName)
	BeatIndexPrefix = EnvOr("BEAT_INDEX_PREFIX", BeatName)
	BeatDescription = EnvOr("BEAT_DESCRIPTION", "")
	BeatVendor      = EnvOr("BEAT_VENDOR", "Elastic")
	BeatLicense     = EnvOr("BEAT_LICENSE", "ASL 2.0")
	BeatURL         = EnvOr("BEAT_URL", "https://www.elastic.co/products/beats/"+BeatName)
	BeatUser        = EnvOr("BEAT_USER", "root")

	Snapshot bool

	versionQualified bool
	versionQualifier string

	FuncMap = map[string]interface{}{
		"beat_doc_branch":   BeatDocBranch,
		"beat_version":      BeatQualifiedVersion,
		"commit":            CommitHash,
		"date":              BuildDate,
		"elastic_beats_dir": ElasticBeatsDir,
		"go_version":        GoVersion,
		"repo":              GetProjectRepoInfo,
		"title":             strings.Title,
		"tolower":           strings.ToLower,
	}
)

func init() {
	if GOOS == "windows" {
		BinaryExt = ".exe"
	}

	var err error
	RaceDetector, err = strconv.ParseBool(EnvOr("RACE_DETECTOR", "false"))
	if err != nil {
		panic(errors.Wrap(err, "failed to parse RACE_DETECTOR env value"))
	}

	TestCoverage, err = strconv.ParseBool(EnvOr("TEST_COVERAGE", "false"))
	if err != nil {
		panic(errors.Wrap(err, "failed to parse TEST_COVERAGE env value"))
	}

	Snapshot, err = strconv.ParseBool(EnvOr("SNAPSHOT", "false"))
	if err != nil {
		panic(errors.Errorf("failed to parse SNAPSHOT env value", err))
	}

	versionQualifier, versionQualified = os.LookupEnv("VERSION_QUALIFIER")
}

// EnvMap returns map containing the common settings variables and all variables
// from the environment. args are appended to the output prior to adding the
// environment variables (so env vars have the highest precedence).
func EnvMap(args ...map[string]interface{}) map[string]interface{} {
	envMap := varMap(args...)

	// Add the environment (highest precedence).
	for _, e := range os.Environ() {
		env := strings.SplitN(e, "=", 2)
		envMap[env[0]] = env[1]
	}

	return envMap
}

func varMap(args ...map[string]interface{}) map[string]interface{} {
	data := map[string]interface{}{
		"GOOS":            GOOS,
		"GOARCH":          GOARCH,
		"GOARM":           GOARM,
		"Platform":        Platform,
		"BinaryExt":       BinaryExt,
		"XPackDir":        XPackDir,
		"BeatName":        BeatName,
		"BeatServiceName": BeatServiceName,
		"BeatIndexPrefix": BeatIndexPrefix,
		"BeatDescription": BeatDescription,
		"BeatVendor":      BeatVendor,
		"BeatLicense":     BeatLicense,
		"BeatURL":         BeatURL,
		"BeatUser":        BeatUser,
		"Snapshot":        Snapshot,
		"Qualifier":       versionQualifier,
	}

	// Add the extra args to the map.
	for _, m := range args {
		for k, v := range m {
			data[k] = v
		}
	}

	return data
}

func dumpVariables() (string, error) {
	var dumpTemplate = `## Variables

GOOS             = {{.GOOS}}
GOARCH           = {{.GOARCH}}
GOARM            = {{.GOARM}}
Platform         = {{.Platform}}
BinaryExt        = {{.BinaryExt}}
XPackDir         = {{.XPackDir}}
BeatName         = {{.BeatName}}
BeatServiceName  = {{.BeatServiceName}}
BeatIndexPrefix  = {{.BeatIndexPrefix}}
BeatDescription  = {{.BeatDescription}}
BeatVendor       = {{.BeatVendor}}
BeatLicense      = {{.BeatLicense}}
BeatURL          = {{.BeatURL}}
BeatUser         = {{.BeatUser}}
VersionQualifier = {{.Qualifier}}

## Functions

beat_doc_branch     = {{ beat_doc_branch }}
beat_version        = {{ beat_version }}
commit              = {{ commit }}
date                = {{ date }}
elastic_beats_dir   = {{ elastic_beats_dir }}
go_version          = {{ go_version }}
repo.RootImportPath = {{ repo.RootImportPath }}
repo.RootDir        = {{ repo.RootDir }}
repo.ImportPath     = {{ repo.ImportPath }}
repo.SubDir         = {{ repo.SubDir }}
`

	return Expand(dumpTemplate)
}

// DumpVariables writes the template variables and values to stdout.
func DumpVariables() error {
	out, err := dumpVariables()
	if err != nil {
		return err
	}

	fmt.Println(out)
	return nil
}

var (
	commitHash     string
	commitHashOnce sync.Once
)

// CommitHash returns the full length git commit hash.
func CommitHash() (string, error) {
	var err error
	commitHashOnce.Do(func() {
		commitHash, err = sh.Output("git", "rev-parse", "HEAD")
	})
	return commitHash, err
}

var (
	elasticBeatsDirValue string
	elasticBeatsDirErr   error
	elasticBeatsDirLock  sync.Mutex
)

// ElasticBeatsDir returns the path to Elastic beats dir.
func ElasticBeatsDir() (string, error) {
	elasticBeatsDirLock.Lock()
	defer elasticBeatsDirLock.Unlock()

	if elasticBeatsDirValue != "" || elasticBeatsDirErr != nil {
		return elasticBeatsDirValue, elasticBeatsDirErr
	}

	elasticBeatsDirValue, elasticBeatsDirErr = findElasticBeatsDir()
	if elasticBeatsDirErr == nil {
		log.Println("Found Elastic Beats dir at", elasticBeatsDirValue)
	}
	return elasticBeatsDirValue, elasticBeatsDirErr
}

// findElasticBeatsDir attempts to find the root of the Elastic Beats directory.
// It checks to see if the current project is elastic/beats, and then if not
// checks the vendor directory.
//
// If your project places the Beats files in a different location (specifically
// the dev-tools/ contents) then you can use SetElasticBeatsDir().
func findElasticBeatsDir() (string, error) {
	repo, err := GetProjectRepoInfo()
	if err != nil {
		return "", err
	}

	if repo.IsElasticBeats() {
		return repo.RootDir, nil
	}

	const devToolsImportPath = elasticBeatsImportPath + "/dev-tools/mage"

	// Search in project vendor directories. Order is relevant
	searchPaths := []string{
		// beats directory of apm-server
		filepath.Join(repo.RootDir, "_beats/dev-tools/vendor"),
		filepath.Join(repo.RootDir, repo.SubDir, "vendor", devToolsImportPath),
		filepath.Join(repo.RootDir, "vendor", devToolsImportPath),
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			return filepath.Join(path, "../.."), nil
		}
	}

	return "", errors.Errorf("failed to find %v in the project's vendor", devToolsImportPath)
}

var (
	buildDate = time.Now().UTC().Format(time.RFC3339)
)

// BuildDate returns the time that the build started.
func BuildDate() string {
	return buildDate
}

var (
	goVersionValue string
	goVersionErr   error
	goVersionOnce  sync.Once
)

// GoVersion returns the version of Go defined in the project's .go-version
// file.
func GoVersion() (string, error) {
	goVersionOnce.Do(func() {
		goVersionValue = os.Getenv("BEAT_GO_VERSION")
		if goVersionValue != "" {
			return
		}

		goVersionValue, goVersionErr = getBuildVariableSources().GetGoVersion()
	})

	return goVersionValue, goVersionErr
}

var (
	beatVersionRegex = regexp.MustCompile(`(?m)^const defaultBeatVersion = "(.+)"\r?$`)
	beatVersionValue string
	beatVersionErr   error
	beatVersionOnce  sync.Once
)

// BeatQualifiedVersion returns the Beat's qualified version.  The value can be overwritten by
// setting VERSION_QUALIFIER in the environment.
func BeatQualifiedVersion() (string, error) {
	version, err := beatVersion()
	if err != nil {
		return "", err
	}
	// version qualifier can intentionally be set to "" to override build time var
	if !versionQualified || versionQualifier == "" {
		return version, nil
	}
	return version + "-" + versionQualifier, nil
}

// BeatVersion returns the Beat's version. The value can be overridden by
// setting BEAT_VERSION in the environment.
func beatVersion() (string, error) {
	beatVersionOnce.Do(func() {
		beatVersionValue = os.Getenv("BEAT_VERSION")
		if beatVersionValue != "" {
			return
		}

		beatVersionValue, beatVersionErr = getBuildVariableSources().GetBeatVersion()
	})

	return beatVersionValue, beatVersionErr
}

var (
	beatDocBranchRegex = regexp.MustCompile(`(?m)doc-branch:\s*([^\s]+)\r?$`)
	beatDocBranchValue string
	beatDocBranchErr   error
	beatDocBranchOnce  sync.Once
)

// BeatDocBranch returns the documentation branch name associated with the
// Beat branch.
func BeatDocBranch() (string, error) {
	beatDocBranchOnce.Do(func() {
		beatDocBranchValue = os.Getenv("BEAT_DOC_BRANCH")
		if beatDocBranchValue != "" {
			return
		}

		beatDocBranchValue, beatDocBranchErr = getBuildVariableSources().GetDocBranch()
	})

	return beatDocBranchValue, beatDocBranchErr
}

// --- BuildVariableSources

var (
	// DefaultBeatBuildVariableSources contains the default locations build
	// variables are read from by Elastic Beats.
	DefaultBeatBuildVariableSources = &BuildVariableSources{
		BeatVersion: "{{ elastic_beats_dir }}/libbeat/version/version.go",
		GoVersion:   "{{ elastic_beats_dir }}/.go-version",
		DocBranch:   "{{ elastic_beats_dir }}/libbeat/docs/version.asciidoc",
	}

	buildVariableSources     *BuildVariableSources
	buildVariableSourcesLock sync.Mutex
)

// SetBuildVariableSources sets the BuildVariableSources that defines where
// certain build data should be sourced from. Community Beats must call this.
func SetBuildVariableSources(s *BuildVariableSources) {
	buildVariableSourcesLock.Lock()
	defer buildVariableSourcesLock.Unlock()

	buildVariableSources = s
}

func getBuildVariableSources() *BuildVariableSources {
	buildVariableSourcesLock.Lock()
	defer buildVariableSourcesLock.Unlock()

	if buildVariableSources != nil {
		return buildVariableSources
	}

	repo, err := GetProjectRepoInfo()
	if err != nil {
		panic(err)
	}
	if repo.IsElasticBeats() {
		buildVariableSources = DefaultBeatBuildVariableSources
		return buildVariableSources
	}

	panic(errors.Errorf("magefile must call mage.SetBuildVariableSources() "+
		"because it is not an elastic beat (repo=%+v)", repo.RootImportPath))
}

// BuildVariableSources is used to explicitly define what files contain build
// variables and how to parse the values from that file. This removes ambiguity
// about where the data is sources and allows a degree of customization for
// community Beats.
//
// Default parsers are used if one is not defined.
type BuildVariableSources struct {
	// File containing the Beat version.
	BeatVersion string

	// Parses the Beat version from the BeatVersion file.
	BeatVersionParser func(data []byte) (string, error)

	// File containing the Go version to be used in cross-builds.
	GoVersion string

	// Parses the Go version from the GoVersion file.
	GoVersionParser func(data []byte) (string, error)

	// File containing the documentation branch.
	DocBranch string

	// Parses the documentation branch from the DocBranch file.
	DocBranchParser func(data []byte) (string, error)
}

func (s *BuildVariableSources) expandVar(in string) (string, error) {
	return expandTemplate("inline", in, map[string]interface{}{
		"elastic_beats_dir": ElasticBeatsDir,
	})
}

// GetBeatVersion reads the BeatVersion file and parses the version from it.
func (s *BuildVariableSources) GetBeatVersion() (string, error) {
	file, err := s.expandVar(s.BeatVersion)
	if err != nil {
		return "", err
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read beat version file=%v", file)
	}

	if s.BeatVersionParser == nil {
		s.BeatVersionParser = parseBeatVersion
	}
	return s.BeatVersionParser(data)
}

// GetGoVersion reads the GoVersion file and parses the version from it.
func (s *BuildVariableSources) GetGoVersion() (string, error) {
	file, err := s.expandVar(s.GoVersion)
	if err != nil {
		return "", err
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read go version file=%v", file)
	}

	if s.GoVersionParser == nil {
		s.GoVersionParser = parseGoVersion
	}
	return s.GoVersionParser(data)
}

// GetDocBranch reads the DocBranch file and parses the branch from it.
func (s *BuildVariableSources) GetDocBranch() (string, error) {
	file, err := s.expandVar(s.DocBranch)
	if err != nil {
		return "", err
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read doc branch file=%v", file)
	}

	if s.DocBranchParser == nil {
		s.DocBranchParser = parseDocBranch
	}
	return s.DocBranchParser(data)
}

func parseBeatVersion(data []byte) (string, error) {
	matches := beatVersionRegex.FindSubmatch(data)
	if len(matches) == 2 {
		return string(matches[1]), nil
	}

	return "", errors.New("failed to parse beat version file")
}

func parseGoVersion(data []byte) (string, error) {
	return strings.TrimSpace(string(data)), nil
}

func parseDocBranch(data []byte) (string, error) {
	matches := beatDocBranchRegex.FindSubmatch(data)
	if len(matches) == 2 {
		return string(matches[1]), nil
	}

	return "", errors.New("failed to parse beat doc branch")
}

// --- ProjectRepoInfo

// ProjectRepoInfo contains information about the project's repo.
type ProjectRepoInfo struct {
	RootImportPath string // Import path at the project root.
	RootDir        string // Root directory of the project.
	ImportPath     string // Import path of the current directory.
	SubDir         string // Relative path from the root dir to the current dir.
}

// IsElasticBeats returns true if the current project is
// github.com/elastic/beats.
func (r *ProjectRepoInfo) IsElasticBeats() bool {
	return r.RootImportPath == elasticBeatsImportPath
}

var (
	repoInfoValue *ProjectRepoInfo
	repoInfoErr   error
	repoInfoOnce  sync.Once
)

// GetProjectRepoInfo returns information about the repo including the root
// import path and the current directory's import path.
func GetProjectRepoInfo() (*ProjectRepoInfo, error) {
	repoInfoOnce.Do(func() {
		repoInfoValue, repoInfoErr = getProjectRepoInfo()
	})

	return repoInfoValue, repoInfoErr
}

func getProjectRepoInfo() (*ProjectRepoInfo, error) {
	var (
		cwd            = CWD()
		rootImportPath string
		srcDir         string
	)

	// Search upward from the CWD to determine the project root based on VCS.
	var errs []string
	for _, gopath := range filepath.SplitList(build.Default.GOPATH) {
		gopath = filepath.Clean(gopath)

		if !strings.HasPrefix(cwd, gopath) {
			// Fixes an issue on macOS when /var is actually /private/var.
			var err error
			gopath, err = filepath.EvalSymlinks(gopath)
			if err != nil {
				errs = append(errs, err.Error())
				continue
			}
		}

		srcDir = filepath.Join(gopath, "src")
		_, root, err := vcs.FromDir(cwd, srcDir)
		if err != nil {
			// Try the next gopath.
			errs = append(errs, err.Error())
			continue
		}
		rootImportPath = root
		break
	}
	if rootImportPath == "" {
		return nil, errors.Errorf("failed to determine root import path (Did "+
			"you git init?, Is the project in the GOPATH? GOPATH=%v, CWD=%v?): %v",
			build.Default.GOPATH, cwd, errs)
	}

	rootDir := filepath.Join(srcDir, rootImportPath)
	subDir, err := filepath.Rel(rootDir, cwd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get relative path to repo root")
	}
	importPath := filepath.ToSlash(filepath.Join(rootImportPath, subDir))

	return &ProjectRepoInfo{
		RootImportPath: rootImportPath,
		RootDir:        rootDir,
		SubDir:         subDir,
		ImportPath:     importPath,
	}, nil
}
