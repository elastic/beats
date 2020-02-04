// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build mage

package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	"github.com/elastic/beats/dev-tools/mage"

	devtools "github.com/elastic/beats/dev-tools/mage"
	"github.com/pkg/errors"

	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/common"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/unittest"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/test"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
)

var hubID = "elastic"
var logDriverName = "elastic-logging-plugin"
var dockerPluginName = filepath.Join(hubID, logDriverName)
var packageStagingDir = "build/package/"
var packageEndDir = "build/distributions/"
var buildDir = filepath.Join(packageStagingDir, logDriverName)
var dockerExportPath = filepath.Join(packageStagingDir, "temproot.tar")
var rootImageName = "rootfsimage"

func init() {
	devtools.BeatLicense = "Elastic License"
	devtools.BeatDescription = "The Docker Logging Driver is a docker plugin for the Elastic Stack."
}

// getPluginName returns the fully qualified name:version string
func getPluginName() (string, error) {
	version, err := mage.BeatQualifiedVersion()
	if err != nil {
		return "", errors.Wrap(err, "error getting beats version")
	}
	return dockerPluginName + ":" + version, nil
}

// createContainer builds the plugin and creates the container that will later become the rootfs used by the plugin
func createContainer(ctx context.Context, cli *client.Client) error {
	goVersion, err := mage.GoVersion()
	if err != nil {
		return errors.Wrap(err, "error determining go version")
	}

	dockerLogBeatDir, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "error getting work dir")
	}

	if !strings.Contains(dockerLogBeatDir, "dockerlogbeat") {
		return errors.Errorf("not in dockerlogbeat directory: %s", dockerLogBeatDir)
	}

	// start to build the root container that'll be used to build the plugin
	tmpDir, err := ioutil.TempDir("", "dockerBuildTar")
	if err != nil {
		return errors.Wrap(err, "Error locating temp dir")
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
		BuildArgs:  map[string]*string{"versionString": &goVersion},
		Tags:       []string{rootImageName},
		Dockerfile: "Dockerfile",
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
	// move back to the x-pack dir
	err = os.Chdir(dockerLogBeatDir)
	if err != nil {
		return errors.Wrap(err, "error returning to dockerlogbeat dir")
	}

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
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return errors.Wrap(err, "Error creating docker client")
	}

	mage.CreateDir(packageStagingDir)
	mage.CreateDir(packageEndDir)
	err = os.MkdirAll(filepath.Join(buildDir, "rootfs"), 0755)
	if err != nil {
		return errors.Wrap(err, "Error creating build dir")
	}

	err = createContainer(ctx, cli)
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
		return errors.Wrap(err, "Error writing exported container")
	}

	//misc prepare operations

	err = mage.Copy("config.json", filepath.Join(buildDir, "config.json"))
	if err != nil {
		return errors.Wrap(err, "error copying config.json")
	}

	// unpack the tar file into a root directory, which is the format needed for the docker plugin create tool
	err = sh.RunV("tar", "-xf", dockerExportPath, "-C", filepath.Join(buildDir, "rootfs"))
	if err != nil {
		return errors.Wrap(err, "error unpacking exported container")
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
	name, err := getPluginName()
	if err != nil {
		return err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return errors.Wrap(err, "Error creating docker client")
	}

	//check to see if we have a plugin we need to remove
	plugins, err := cli.PluginList(ctx, filters.Args{})
	if err != nil {
		return errors.Wrap(err, "error getting list of plugins")
	}
	oursExists := false
	for _, plugin := range plugins {
		if strings.Contains(plugin.Name, logDriverName) {
			oursExists = true
		}
	}
	if oursExists {
		err = cli.PluginDisable(ctx, name, types.PluginDisableOptions{Force: true})
		if err != nil {
			return errors.Wrap(err, "error disabling plugin")
		}
		err = cli.PluginRemove(ctx, name, types.PluginRemoveOptions{Force: true})
		if err != nil {
			return errors.Wrap(err, "error removing plugin")
		}
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

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return errors.Wrap(err, "Error creating docker client")
	}

	archiveOpts := &archive.TarOptions{
		Compression:  archive.Uncompressed,
		IncludeFiles: []string{"rootfs", "config.json"},
	}
	archive, err := archive.TarWithOptions(buildDir, archiveOpts)
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

// Export exports a "ready" root filesystem and config.json into a tarball
func Export() error {
	version, err := mage.BeatQualifiedVersion()
	if err != nil {
		return errors.Wrap(err, "error getting beats version")
	}

	if mage.Snapshot {
		version = version + "-SNAPSHOT"
	}

	tarballName := fmt.Sprintf("%s-%s-%s.tar.gz", logDriverName, version, "docker-plugin")

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

	return nil
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	return devtools.CrossBuild(devtools.ForPlatforms("linux/amd64"))
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
	buildArgs.OutputDir = "build"
	return devtools.GolangCrossBuild(buildArgs)
}

// Package builds a "release" tarball that can be used later with `docker plugin create`
func Package() {
	mg.SerialDeps(Build, Export)
}

// BuildAndInstall builds and installs the plugin
func BuildAndInstall() {
	mg.SerialDeps(Build, Install)
}

// IntegTest is currently a dummy test for the `testsuite` target
func IntegTest() {
	fmt.Printf("There are no Integration tests for The Elastic Log Plugin\n")
}

// Update is currently a dummy test for the `testsuite` target
func Update() {
	fmt.Printf("There is no Update for The Elastic Log Plugin\n")
}
