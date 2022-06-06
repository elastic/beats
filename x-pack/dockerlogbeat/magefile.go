// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build mage
// +build mage

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"

	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/common"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/unittest"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/integtest/notests"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/test"
)

const (
	hubID             = "elastic"
	logDriverName     = "elastic-logging-plugin"
	dockerPluginName  = hubID + "/" + logDriverName
	packageStagingDir = "build/package/"
	packageEndDir     = "build/distributions/"
	rootImageName     = "rootfsimage"
	dockerfileTmpl    = "Dockerfile.tmpl"
)

var (
	buildDir         = filepath.Join(packageStagingDir, logDriverName)
	dockerExportPath = filepath.Join(packageStagingDir, "temproot.tar")

	platformMap = map[string]map[string]interface{}{
		"amd64": map[string]interface{}{
			"from": "alpine:3.10",
		},
		"arm64": map[string]interface{}{
			"from": "arm64v8/alpine:3.10",
		},
	}
)

func init() {
	devtools.BeatLicense = "Elastic License"
	devtools.BeatDescription = "The Docker Logging Driver is a docker plugin for the Elastic Stack."
	devtools.Platforms = devtools.Platforms.Filter("linux/amd64 linux/arm64")
}

// getPluginName returns the fully qualified name:version string.
func getPluginName() (string, error) {
	version, err := devtools.BeatQualifiedVersion()
	if err != nil {
		return "", errors.Wrap(err, "error getting beats version")
	}
	return dockerPluginName + ":" + version, nil
}

// createContainer builds the plugin and creates the container that will later become the rootfs used by the plugin
func createContainer(ctx context.Context, cli *client.Client, arch string) error {
	dockerLogBeatDir, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "error getting work dir")
	}

	if !strings.Contains(dockerLogBeatDir, "dockerlogbeat") {
		return errors.Errorf("not in dockerlogbeat directory: %s", dockerLogBeatDir)
	}

	dockerfile := filepath.Join(packageStagingDir, "Dockerfile")
	err = devtools.ExpandFile(dockerfileTmpl, dockerfile, platformMap[arch])
	if err != nil {
		return errors.Wrap(err, "error while expanding Dockerfile template")
	}

	// start to build the root container that'll be used to build the plugin
	tmpDir, err := ioutil.TempDir("", "dockerBuildTar")
	if err != nil {
		return errors.Wrap(err, "error locating temp dir")
	}
	defer sh.Rm(tmpDir)

	tarPath := filepath.Join(tmpDir, "tarRoot.tar")
	err = sh.RunV("tar", "cf", tarPath, "./")
	if err != nil {
		return errors.Wrap(err, "error creating tar")
	}

	buildContext, err := os.Open(tarPath)
	if err != nil {
		return errors.Wrap(err, "error opening temp dur")
	}
	defer buildContext.Close()

	buildOpts := types.ImageBuildOptions{
		Tags:       []string{rootImageName},
		Dockerfile: dockerfile,
	}
	//build, wait for output
	buildResp, err := cli.ImageBuild(ctx, buildContext, buildOpts)
	if err != nil {
		return errors.Wrap(err, "error building final container image")
	}
	defer buildResp.Body.Close()
	// This blocks until the build operation completes
	buildStr, errBufRead := ioutil.ReadAll(buildResp.Body)
	if errBufRead != nil {
		return errors.Wrap(err, "error reading from docker output")
	}
	fmt.Printf("%s\n", string(buildStr))

	return nil
}

// BuildContainer builds docker rootfs container root
// There's a somewhat complicated process for this:
// * Create a container to build the plugin itself
// * Copy that to a bare-bones container that will become the runc container used by docker
// * Export that container
// * Unpack the tar from the exported container
// * send this to the plugin create API endpoint
func BuildContainer(ctx context.Context) error {
	// setup
	cli, err := newDockerClient(ctx)
	if err != nil {
		return errors.Wrap(err, "error creating docker client")
	}

	devtools.CreateDir(packageStagingDir)
	devtools.CreateDir(packageEndDir)
	err = os.MkdirAll(filepath.Join(buildDir, "rootfs"), 0755)
	if err != nil {
		return errors.Wrap(err, "error creating build dir")
	}

	for _, plat := range devtools.Platforms {
		arch := plat.GOARCH()
		if runtime.GOARCH != arch {
			fmt.Println("Skippping building for", arch, "as runtime is different")
			continue
		}

		err = createContainer(ctx, cli, arch)
		if err != nil {
			return errors.Wrap(err, "error creating base container")
		}

		// create the container that will become our rootfs
		CreatedContainerBody, err := cli.ContainerCreate(ctx, &container.Config{Image: rootImageName}, nil, nil, "")
		if err != nil {
			return errors.Wrap(err, "error creating container")
		}

		defer func() {
			// cleanup
			if _, noClean := os.LookupEnv("DOCKERLOGBEAT_NO_CLEANUP"); !noClean {
				err = cleanDockerArtifacts(ctx, CreatedContainerBody.ID, cli)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error cleaning up docker: %s", err)
				}
			}
		}()

		fmt.Printf("Got image: %#v\n", CreatedContainerBody.ID)

		file, err := os.Create(dockerExportPath)
		if err != nil {
			return errors.Wrap(err, "error creating tar archive")
		}

		// export the container to a tar file
		exportReader, err := cli.ContainerExport(ctx, CreatedContainerBody.ID)
		if err != nil {
			return errors.Wrap(err, "error exporting container")
		}

		_, err = io.Copy(file, exportReader)
		if err != nil {
			return errors.Wrap(err, "error writing exported container")
		}

		//misc prepare operations

		err = devtools.Copy("config.json", filepath.Join(buildDir, "config.json"))
		if err != nil {
			return errors.Wrap(err, "error copying config.json")
		}

		// unpack the tar file into a root directory, which is the format needed for the docker plugin create tool
		err = sh.RunV("tar", "-xf", dockerExportPath, "-C", filepath.Join(buildDir, "rootfs"))
		if err != nil {
			return errors.Wrap(err, "error unpacking exported container")
		}
	}

	return nil
}

func cleanDockerArtifacts(ctx context.Context, containerID string, cli *client.Client) error {
	fmt.Printf("Removing container %s\n", containerID)
	err := cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{RemoveVolumes: true, Force: true})
	if err != nil {
		return errors.Wrap(err, "error removing container")
	}

	resp, err := cli.ImageRemove(ctx, rootImageName, types.ImageRemoveOptions{Force: true})
	if err != nil {
		return errors.Wrap(err, "error removing image")
	}
	fmt.Printf("Removed image: %#v\n", resp)
	return nil
}

// Uninstall removes working objects and containers
func Uninstall(ctx context.Context) error {
	cli, err := newDockerClient(ctx)
	if err != nil {
		return errors.Wrap(err, "error creating docker client")
	}

	//check to see if we have a plugin we need to remove
	plugins, err := cli.PluginList(ctx, filters.Args{})
	if err != nil {
		return errors.Wrap(err, "error getting list of plugins")
	}

	toRemoveName := ""
	for _, plugin := range plugins {
		if strings.Contains(plugin.Name, logDriverName) {
			toRemoveName = plugin.Name
			break
		}
	}
	if toRemoveName == "" {
		return nil
	}

	err = cli.PluginDisable(ctx, toRemoveName, types.PluginDisableOptions{Force: true})
	if err != nil {
		return errors.Wrap(err, "error disabling plugin")
	}
	err = cli.PluginRemove(ctx, toRemoveName, types.PluginRemoveOptions{Force: true})
	if err != nil {
		return errors.Wrap(err, "error removing plugin")
	}

	return nil
}

// Install installs the plugin
func Install(ctx context.Context) error {
	mg.Deps(Uninstall)
	if _, err := os.Stat(filepath.Join(packageStagingDir, "rootfs")); os.IsNotExist(err) {
		mg.Deps(Build)
	}

	name, err := getPluginName()
	if err != nil {
		return err
	}

	cli, err := newDockerClient(ctx)
	if err != nil {
		return errors.Wrap(err, "error creating docker client")
	}

	archive, err := tar(buildDir, "rootfs", "config.json")
	if err != nil {
		return errors.Wrap(err, "error creating archive of work dir")
	}

	err = cli.PluginCreate(ctx, archive, types.PluginCreateOptions{RepoName: name})
	if err != nil {
		return errors.Wrap(err, "error creating plugin")
	}

	err = cli.PluginEnable(ctx, name, types.PluginEnableOptions{})
	if err != nil {
		return errors.Wrap(err, "error enabling plugin")
	}

	return nil
}

func tar(dir string, files ...string) (io.Reader, error) {
	var archive bytes.Buffer
	var stdErr bytes.Buffer
	args := append([]string{"-C", dir, "-cf", "-"}, files...)
	_, err := sh.Exec(nil, &archive, &stdErr, "tar", args...)
	if err != nil {
		return nil, errors.Wrap(err, stdErr.String())
	}

	return &archive, nil
}

// Export exports a "ready" root filesystem and config.json into a tarball
func Export() error {
	version, err := devtools.BeatQualifiedVersion()
	if err != nil {
		return errors.Wrap(err, "error getting beats version")
	}

	if devtools.Snapshot {
		version = version + "-SNAPSHOT"
	}

	for _, plat := range devtools.Platforms {
		arch := plat.GOARCH()
		tarballName := fmt.Sprintf("%s-%s-%s-%s.tar.gz", logDriverName, version, "docker-plugin", arch)

		outpath := filepath.Join("../..", packageEndDir, tarballName)

		err = os.Chdir(packageStagingDir)
		if err != nil {
			return errors.Wrap(err, "error changing directory")
		}

		err = sh.RunV("tar", "zcf", outpath,
			filepath.Join(logDriverName, "rootfs"),
			filepath.Join(logDriverName, "config.json"))
		if err != nil {
			return errors.Wrap(err, "error creating release tarball")
		}
		return errors.Wrap(devtools.CreateSHA512File(outpath), "failed to create .sha512 file")
	}

	return nil
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	return devtools.CrossBuild()
}

// Build builds the base container used by the docker plugin
func Build() {
	mg.SerialDeps(CrossBuild, BuildContainer)
}

// GolangCrossBuild build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
	buildArgs := devtools.DefaultBuildArgs()
	buildArgs.CGO = false
	buildArgs.Static = true
	buildArgs.OutputDir = "build/plugin"
	return devtools.GolangCrossBuild(buildArgs)
}

// Package builds a "release" tarball that can be used later with `docker plugin create`
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	if !isSupportedPlatform() {
		fmt.Println(">> package: skipping because no supported platform is enabled")
		return
	}

	mg.SerialDeps(Build, Export)
}

func isSupportedPlatform() bool {
	_, isAMD64Selected := devtools.Platforms.Get("linux/amd64")
	_, isARM64Selected := devtools.Platforms.Get("linux/arm64")
	arch := runtime.GOARCH

	if arch == "amd64" && isARM64Selected {
		devtools.Platforms = devtools.Platforms.Remove("linux/arm64")
	} else if arch == "arm64" && isAMD64Selected {
		devtools.Platforms = devtools.Platforms.Remove("linux/amd64")
	}

	return len(devtools.Platforms) > 0
}

// BuildAndInstall builds and installs the plugin
func BuildAndInstall() {
	mg.SerialDeps(Build, Install)
}

// Update is currently a dummy test for the `testsuite` target
func Update() {
	fmt.Println(">> update: There is no Update for The Elastic Log Plugin")
}

func newDockerClient(ctx context.Context) (*client.Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	cli.NegotiateAPIVersion(ctx)
	return cli, nil
}
