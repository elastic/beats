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
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/file"
)

const defaultCrossBuildTarget = "golangCrossBuild"

// Platforms contains the set of target platforms for cross-builds. It can be
// modified at runtime by setting the PLATFORMS environment variable.
// See NewPlatformList for details about platform filtering expressions.
var Platforms = BuildPlatforms.Defaults()

func init() {
	// Allow overriding via PLATFORMS.
	if expression := os.Getenv("PLATFORMS"); len(expression) > 0 {
		Platforms = NewPlatformList(expression)
	}
}

// CrossBuildOption defines a option to the CrossBuild target.
type CrossBuildOption func(params *crossBuildParams)

// ImageSelectorFunc returns the name of the builder image.
type ImageSelectorFunc func(platform string) (string, error)

// ForPlatforms filters the platforms based on the given expression.
func ForPlatforms(expr string) func(params *crossBuildParams) {
	return func(params *crossBuildParams) {
		params.Platforms = params.Platforms.Filter(expr)
	}
}

// WithTarget specifies the mage target to execute inside the golang-crossbuild
// container.
func WithTarget(target string) func(params *crossBuildParams) {
	return func(params *crossBuildParams) {
		params.Target = target
	}
}

// InDir specifies the base directory to use when cross-building.
func InDir(path ...string) func(params *crossBuildParams) {
	return func(params *crossBuildParams) {
		params.InDir = filepath.Join(path...)
	}
}

// Serially causes each cross-build target to be executed serially instead of
// in parallel.
func Serially() func(params *crossBuildParams) {
	return func(params *crossBuildParams) {
		params.Serial = true
	}
}

// ImageSelector returns the name of the selected builder image.
func ImageSelector(f ImageSelectorFunc) func(params *crossBuildParams) {
	return func(params *crossBuildParams) {
		params.ImageSelector = f
	}
}

// AddPlatforms sets dependencies on others platforms.
func AddPlatforms(expressions ...string) func(params *crossBuildParams) {
	return func(params *crossBuildParams) {
		var list BuildPlatformList
		for _, expr := range expressions {
			list = NewPlatformList(expr)
			params.Platforms = params.Platforms.Merge(list)
		}
	}
}

type crossBuildParams struct {
	Platforms     BuildPlatformList
	Target        string
	Serial        bool
	InDir         string
	ImageSelector ImageSelectorFunc
}

// CrossBuild executes a given build target once for each target platform.
func CrossBuild(options ...CrossBuildOption) error {
	params := crossBuildParams{Platforms: Platforms, Target: defaultCrossBuildTarget, ImageSelector: crossBuildImage}
	for _, opt := range options {
		opt(&params)
	}

	// Docker is required for this target.
	if err := HaveDocker(); err != nil {
		return err
	}

	if len(params.Platforms) == 0 {
		log.Printf("Skipping cross-build of target=%v because platforms list is empty.", params.Target)
		return nil
	}

	// Build the magefile for Linux so we can run it inside the container.
	mg.Deps(buildMage)

	log.Println("crossBuild: Platform list =", params.Platforms)
	var deps []interface{}
	for _, buildPlatform := range params.Platforms {
		if !buildPlatform.Flags.CanCrossBuild() {
			return fmt.Errorf("unsupported cross build platform %v", buildPlatform.Name)
		}
		builder := GolangCrossBuilder{buildPlatform.Name, params.Target, params.InDir, params.ImageSelector}
		if params.Serial {
			if err := builder.Build(); err != nil {
				return errors.Wrapf(err, "failed cross-building target=%v for platform=%v %v", params.ImageSelector,
					params.Target, buildPlatform.Name)
			}
		} else {
			deps = append(deps, builder.Build)
		}
	}

	// Each build runs in parallel.
	Parallel(deps...)
	return nil
}

// CrossBuildXPack executes the 'golangCrossBuild' target in the Beat's
// associated x-pack directory to produce a version of the Beat that contains
// Elastic licensed content.
func CrossBuildXPack(options ...CrossBuildOption) error {
	o := []CrossBuildOption{InDir("x-pack", BeatName)}
	o = append(o, options...)
	return CrossBuild(o...)
}

// buildMage pre-compiles the magefile to a binary using the native GOOS/GOARCH
// values for Docker. It has the benefit of speeding up the build because the
// mage -compile is done only once rather than in each Docker container.
func buildMage() error {
	return sh.Run("mage", "-f", "-goos=linux", "-goarch=amd64",
		"-compile", CreateDir(filepath.Join("build", "mage-linux-amd64")))
}

func crossBuildImage(platform string) (string, error) {
	tagSuffix := "main"

	switch {
	case strings.HasPrefix(platform, "darwin"):
		tagSuffix = "darwin"
	case strings.HasPrefix(platform, "linux/arm"):
		tagSuffix = "arm"
	case strings.HasPrefix(platform, "linux/mips"):
		tagSuffix = "mips"
	case strings.HasPrefix(platform, "linux/ppc"):
		tagSuffix = "ppc"
	case platform == "linux/s390x":
		tagSuffix = "s390x"
	case strings.HasPrefix(platform, "linux"):
		// Use an older version of libc to gain greater OS compatibility.
		// Debian 7 uses glibc 2.13.
		tagSuffix = "main-debian7"
	}

	goVersion, err := GoVersion()
	if err != nil {
		return "", err
	}

	return BeatsCrossBuildImage + ":" + goVersion + "-" + tagSuffix, nil
}

// GolangCrossBuilder executes the specified mage target inside of the
// associated golang-crossbuild container image for the platform.
type GolangCrossBuilder struct {
	Platform      string
	Target        string
	InDir         string
	ImageSelector ImageSelectorFunc
}

// Build executes the build inside of Docker.
func (b GolangCrossBuilder) Build() error {
	fmt.Printf(">> %v: Building for %v\n", b.Target, b.Platform)

	repoInfo, err := GetProjectRepoInfo()
	if err != nil {
		return errors.Wrap(err, "failed to determine repo root and package sub dir")
	}

	mountPoint := filepath.ToSlash(filepath.Join("/go", "src", repoInfo.RootImportPath))
	// use custom dir for build if given, subdir if not:
	cwd := repoInfo.SubDir
	if b.InDir != "" {
		cwd = b.InDir
	}
	workDir := filepath.ToSlash(filepath.Join(mountPoint, cwd))

	buildCmd, err := filepath.Rel(workDir, filepath.Join(mountPoint, repoInfo.SubDir, "build/mage-linux-amd64"))
	if err != nil {
		return errors.Wrap(err, "failed to determine mage-linux-amd64 relative path")
	}

	dockerRun := sh.RunCmd("docker", "run")
	image, err := b.ImageSelector(b.Platform)
	if err != nil {
		return errors.Wrap(err, "failed to determine golang-crossbuild image tag")
	}
	verbose := ""
	if mg.Verbose() {
		verbose = "true"
	}
	var args []string
	if runtime.GOOS != "windows" {
		args = append(args,
			"--env", "EXEC_UID="+strconv.Itoa(os.Getuid()),
			"--env", "EXEC_GID="+strconv.Itoa(os.Getgid()),
		)
	}
	if versionQualified {
		args = append(args, "--env", "VERSION_QUALIFIER="+versionQualifier)
	}
	args = append(args,
		"--rm",
		"--env", "MAGEFILE_VERBOSE="+verbose,
		"--env", "MAGEFILE_TIMEOUT="+EnvOr("MAGEFILE_TIMEOUT", ""),
		"-v", repoInfo.RootDir+":"+mountPoint,
		"-w", workDir,
		image,
		"--build-cmd", buildCmd+" "+b.Target,
		"-p", b.Platform,
	)

	return dockerRun(args...)
}

// DockerChown chowns files generated during build. EXEC_UID and EXEC_GID must
// be set in the containers environment otherwise this is a noop.
func DockerChown(path string) {
	// Chown files generated during build that are root owned.
	uid, _ := strconv.Atoi(EnvOr("EXEC_UID", "-1"))
	gid, _ := strconv.Atoi(EnvOr("EXEC_GID", "-1"))
	if uid > 0 && gid > 0 {
		log.Printf(">>> Fixing file ownership issues from Docker at path=%v", path)
		if err := chownPaths(uid, gid, path); err != nil {
			log.Println(err)
		}
	}
}

// chownPaths will chown the file and all of the dirs specified in the path.
func chownPaths(uid, gid int, path string) error {
	start := time.Now()
	defer log.Printf("chown took: %v", time.Now().Sub(start))

	return filepath.Walk(path, func(name string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get the file's UID and GID.
		stat, err := file.Wrap(info)
		if err != nil {
			return err
		}
		fileUID, _ := stat.UID()
		fileGID, _ := stat.GID()
		if uid == fileUID && gid == fileGID {
			// Skip if UID/GID are already a match.
			return nil
		}

		log.Printf("chown file: %v", name)
		if err := os.Chown(name, uid, gid); err != nil {
			return errors.Wrapf(err, "failed to chown path=%v", name)
		}
		return nil
	})
}
