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
func createContainer(cli *client.Client) error {
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

	// change back to the root beats dir so we can send the proper build context to docker
	err = os.Chdir("../..")
	if err != nil {
		return errors.Wrap(err, "error changing directory")
	}

	// start to build the root container that'll be used to build the plugin
	tmpDir, err := ioutil.TempDir("", "dockerBuildTar")
	if err != nil {
		return errors.Wrap(err, "Error locating temp dir")
	}
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
		Target:     "final",
		Tags:       []string{rootImageName},
		Dockerfile: "x-pack/dockerlogbeat/Dockerfile",
	}
	//build, wait for output
	buildResp, err := cli.ImageBuild(context.Background(), buildContext, buildOpts)
	defer buildResp.Body.Close()
	// buf := new(bytes.Buffer)
	// _, errBufRead := buf.ReadFrom(buildResp.Body)
	buf, errBufRead := ioutil.ReadAll(buildResp.Body)
	if errBufRead != nil {
		return errors.Wrap(err, "error reading from docker output")
	}
	if err != nil {
		fmt.Printf("Docker response: \n %s\n", string(buf))
		return errors.Wrap(err, "error building final container image")
	}

	// move back to the x-pack dir
	err = os.Chdir(dockerLogBeatDir)
	if err != nil {
		return errors.Wrap(err, "error returning to dockerlogbeat dir")
	}

	err = sh.Rm(tmpDir)
	if err != nil {
		return errors.Wrap(err, "error removing temp dir")
	}

	return nil
}

// Build builds docker rootfs container root
// There's a somewhat complicated process for this:
// * Create a container to build the plugin itself
// * Copy that to a bare-bones container that will become the runc container used by docker
// * Export that container
// * Unpack the tar from the exported container
// * send this to the plugin create API endpoint
func Build() error {
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

	err = createContainer(cli)
	if err != nil {
		return errors.Wrap(err, "error creating base container")
	}

	// create the container that will become our rootfs
	CreatedContainerBody, err := cli.ContainerCreate(context.Background(), &container.Config{Image: rootImageName}, nil, nil, "")
	if err != nil {
		return errors.Wrap(err, "error creating container")
	}

	defer func() {
		// cleanup
		if _, noClean := os.LookupEnv("DOCKERLOGBEAT_NO_CLEANUP"); !noClean {
			err = cleanDockerArtifacts(CreatedContainerBody.ID, cli)
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
	exportReader, err := cli.ContainerExport(context.Background(), CreatedContainerBody.ID)
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

func cleanDockerArtifacts(containerID string, cli *client.Client) error {
	fmt.Printf("Removing container %s\n", containerID)
	err := cli.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{RemoveVolumes: true, Force: true})
	if err != nil {
		return errors.Wrap(err, "error removing container")
	}

	resp, err := cli.ImageRemove(context.Background(), rootImageName, types.ImageRemoveOptions{Force: true})
	if err != nil {
		return errors.Wrap(err, "error removing image")
	}
	fmt.Printf("Removed image: %#v\n", resp)
	return nil
}

// Uninstall removes working objects and containers
func Uninstall() error {
	name, err := getPluginName()
	if err != nil {
		return err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return errors.Wrap(err, "Error creating docker client")
	}

	//check to see if we have a plugin we need to remove
	plugins, err := cli.PluginList(context.Background(), filters.Args{})
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
		err = cli.PluginDisable(context.Background(), name, types.PluginDisableOptions{Force: true})
		if err != nil {
			return errors.Wrap(err, "error disabling plugin")
		}
		err = cli.PluginRemove(context.Background(), name, types.PluginRemoveOptions{Force: true})
		if err != nil {
			return errors.Wrap(err, "error removing plugin")
		}
	}

	return nil
}

// Install installs the plugin
func Install() error {
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

	err = cli.PluginCreate(context.Background(), archive, types.PluginCreateOptions{RepoName: name})
	if err != nil {
		return errors.Wrap(err, "error creating plugin")
	}

	err = cli.PluginEnable(context.Background(), name, types.PluginEnableOptions{})
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
