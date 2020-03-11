#!/usr/bin/env bash


#///////////////////////////////////////////
# Global Variables
#///////////////////////////////////////////

PLUGIN_NAME="elastic/elastic-logging-plugin"
CONTAINER_ARTIFACT_NAME="ossifrage/rootfsimage"
CONTAINER_NAME="plugin_container"
ROOTFS_EXPORT_NAME="tmp_out.tar"
ROOTFS_NAME="rootfs"
VERSION=""
PLUGIN_FULL=""
TARGET_TO_COPY=""


#///////////////////////////////////////////
# Helper Functions
#///////////////////////////////////////////


cleanup(){
    echo "Cleaning up..."
    if [ -e "$ROOTFS_EXPORT_NAME" ]; then
    rm $ROOTFS_EXPORT_NAME
    fi
    if [ -e "$ROOTFS_NAME" ]; then
    rm -rf $ROOTFS_NAME
    fi
    if [ -e "config.json" ]; then
    rm config.json
    fi
    docker container rm $CONTAINER_NAME
    docker image rm $CONTAINER_ARTIFACT_NAME
}

# cli_help prints  --help
cli_help(){
    cli_name=${0##*/}

    echo "
$cli_name
Utility to import file into the Elastic Log Driver

Usage: $cli_name [FILE OR DIRECTORY]
"
exit 1
}

# run_docker_cmds performs the docker operations to pull and export the container
run_docker_cmds(){

    echo "Downloading container..."
    docker pull -q $CONTAINER_ARTIFACT_NAME


    echo "Creating container..."
    docker container create --name plugin_container $CONTAINER_ARTIFACT_NAME

    echo "Exporting...."
    mkdir -p $ROOTFS_NAME
    docker export -o $ROOTFS_EXPORT_NAME $CONTAINER_NAME
}

# build_plugin builds the plugin from the tar file produced by docker export
build_plugin(){
    echo "Building plugin...."

    tar xf $ROOTFS_EXPORT_NAME -C rootfs

    echo "Adding $TARGET_TO_COPY..."

    cp -r "$TARGET_TO_COPY" "$ROOTFS_NAME"

    echo "Creating new plugin..."
    wget -q https://raw.githubusercontent.com/elastic/beats/master/x-pack/dockerlogbeat/config.json 
    docker plugin create "$PLUGIN_FULL" . 
    docker plugin enable "$PLUGIN_FULL"
}

# input_checks verifies we have a valid CLI argument
input_checks(){
    # Check args
    if [ $# -eq 0 ]; then
        cli_help
    fi
    # Check if file exists
    if [ ! -e "$1" ]; then
        echo "$1 does not exist!"
        exit 1
    fi
    TARGET_TO_COPY=$1
    echo "Will copy $TARGET_TO_COPY into plugin."
}

check_plugins(){
    # Check to see if we have a plugin installed.
    # If we do, grab the name, then remove it
    PLUGINS=$(docker plugin list --format '{{.Name}}' | grep $PLUGIN_NAME)


    if [ -z "$PLUGINS" ]
    then
        VERSION="8.0.0"
        PLUGIN_FULL=$PLUGIN_NAME:$VERSION
        echo "No plugins found. New plugin will be $PLUGIN_FULL"
    else
        PLUGIN_FULL=$PLUGINS
        echo "Plugin found at $PLUGIN_FULL..."
        echo "Removing old plugin..."
        docker plugin disable "$PLUGIN_FULL"
        docker plugin remove "$PLUGIN_FULL"
    fi
}

#///////////////////////////////////////////
# Begin Script
#///////////////////////////////////////////

# Setup
input_checks $*

check_plugins

# End setup
set -e

run_docker_cmds

build_plugin

if [ -z "$BUNDLE_NO_CLEANUP" ]; then
    cleanup
fi