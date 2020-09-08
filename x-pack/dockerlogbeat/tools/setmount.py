#!/usr/bin/env python3

"""Manually build a docker plugin via a custom script"""

import requests
import sys
import tempfile
import os
import io
import tarfile
import json
import re
import subprocess
import shutil
import argparse


PLUGIN_URI = 'elastic/elastic-logging-plugin'
API_URL = 'https://registry-1.docker.io/v2'

USAGE = """
Example Usage:

./setmount.py /path/to/certs/on/docker/host

You can also specify a custom name and version tag for the rebuild plugin via `--tag` and `--name`. By default, the rebuilt plugin will iterate the patch release of the latest plugin.
By defualt, this script will build a new plugin based on the latest release available from the docker store. To specify a different version, use the `--from_release` argument.

This script rebuilds a plugin and adds a custom bindmount path. This can be used to add user-supplied certificates to a Plugin via a mounted directory. 
The host directory will be mounted as `/usr/local/share/ca-certificates` inside the plugin.

After the bindmount has been created, you can adjust the bindmount source and destination via `docker plugin set`:

$ docker plugin set elastic/elastic-logging-plugin:8.0.0 CUSTOM_DIR.source="/new/source/bindmount"

In cases where you're building the plugin on a system that doesn't have access to the docker daemon where you want to run the plugin, 
you can either use `docker plugin push` to push to your own docker repo (note that you must change the name with `--name`), or use the `--to_tar` flag with a target directory to generate a tarball.
For example: `./setmount.py --to_tar /path/to/write/tar/new_plugin.tar.gz /path/to/certs/on/docker/host`
To generate a docker plugin from the tarball, unpack the `.tar.gz` archive, and then run `docker plugin create NEW_PLUGIN_NAME PATH_TO_UNPACKED_DIRECTORY`.
"""

def cleanup(tmpfs: str):
    """cleanup our tmpdir"""
    print("cleaning up", tmpfs)
    shutil.rmtree(tmpfs)

def build_archive(tmpfs: str, target: str, name: str, tag: str):
    """build a tar.gz archive of the plugin build manifest"""
    print("Opening tar file.")
    tar = tarfile.open(target, "w:gz")
    print("Creating tar archive.")
    tar.add(tmpfs + "/config.json", arcname="archive/config.json")
    tar.add(tmpfs + "/rootfs", arcname="archive/rootfs")
    tar.close()
    print("====================================")
    print("A tar archive has been created at ", target)
    print("To build the plugin from the archive, unpack the tar.gz archive and run")
    print("docker plugin create {}:{} archive/".format(name, tag))
    print("====================================")
    cleanup(tmpfs)


def rebuild_plugin(tmpfs: str, name: str, tag: str):
    """Rebuild the plugin via docker"""

    # so we don't collide we anything else, iterate the semver string
    reg = re.compile(r'''(\d+)\.(\d+)\.(\d+)''')
    semver = list(reg.findall(tag))[0]
    if len(semver) is not 3:
        print("Got bad tag, expecting semver: ", tag)
        sys.exit(1)
    patch = int(semver[-1]) + 1

    release_as = str(semver[0]) + "." + str(semver[1]) + "." + str(patch)
    new_name = name + ":" + release_as
    print("Building new plugin...")
    try:
        create = subprocess.run(["docker", "plugin", "create", new_name, tmpfs], capture_output=True, check=True)
        plugin_name = create.stdout.decode("utf-8").strip()
        print("====================================")
        print("A new plugin has been created as {}.".format(plugin_name))
        print("Run 'docker plugin enable {}' to enable the plugin.".format(plugin_name))
        print("You can also run 'docker plugin inspect {}' in view the new mount settings.".format(plugin_name))
        print("====================================")
    except subprocess.CalledProcessError as plugin_err:
        print("Error creating plugin: ", plugin_err)
        print("Output: ", plugin_err.stdout, plugin_err.stderr)
        cleanup(tmpfs)
        sys.exit(1)

    cleanup(tmpfs)


def setup_create(config: dict, rootfs_raw: bytes, mount_source: str):
    """setup_create writes the rootfs and config needed for docker plugin create"""

    tempdir = tempfile.mkdtemp()
    rootfs_path = tempdir + "/rootfs"
    config_path = tempdir + "/config.json"
    os.mkdir(rootfs_path, 755)
    print("Created tmpdir at ", tempdir)

    file_obj = io.BytesIO(rootfs_raw)
    print("Extracting tar archive to ", rootfs_path)
    rootfs_tar = tarfile.open(fileobj=file_obj, mode="r:gz")
    rootfs_tar.extractall(rootfs_path)

    insert_config = {
        "name": "CUSTOM_DIR",
        "description": "Mount for custom certs installed via setmount.py",
        "destination": "/usr/local/share/ca-certificates",
        "source": mount_source,
        "type": "none",
        "options": [
            "rw",
            "rbind"
        ],
        "Settable": [
            "source",
            "destination"
        ]
    }

    # In case someone has a pre-7.9 version
    if "mounts" in config:
        config["mounts"].appendj(insert_config)
    else:
        config["mounts"] = [insert_config]

    print("Writing new config.json to ", config_path)
    with open(config_path, 'w') as outfile:
        json.dump(config, outfile)

    return tempdir

def setup_session():
    """Setup a stateful session to the registry."""

    session = requests.Session()
    # This first API check is mostly a formality to make sure we have the right API and auth
    api_resp = session.get(API_URL)
    if api_resp.status_code == 200:
        return session
    if api_resp.status_code == 404:
        print("v2 API not supported, see https://docs.docker.com/registry/spec/api/#api-version-check")
        sys.exit(1)
    if api_resp.status_code == 401:
        # the auth request realm and service domain are part of the Www-Authenticate header.
        # Extract them and send the auth properly.
        auth_reqest = api_resp.headers["Www-Authenticate"]
        reg = re.compile(r'''(\w+)[=] ?"?([\w/:\.]*)"?''')
        auth_metadata = dict(reg.findall(auth_reqest))
        realm = auth_metadata["realm"]
        service = auth_metadata["service"]
        token_api = realm + "?scope=repository(plugin):" + PLUGIN_URI + ":pull&service=" + service
        # Actual auth call
        resp = session.get(token_api)
        resp.raise_for_status()

        token = resp.json()["token"]
        print("Got session token ", token[0:20])
        session.headers.update({"Authorization": "Bearer " + token})

    return session

def get_tag(session: requests.Session, from_release: str):
    """Get the correct tag. 'latest' doesn't seem to play nice with plugins."""

    tags_url = API_URL + "/" + PLUGIN_URI +"/tags/list"
    tags_resp = session.get(tags_url)
    tags_resp.raise_for_status()
    tags = tags_resp.json()["tags"]
    if from_release in tags:
        return from_release
    if from_release is not "" and from_release not in tags:
        print("Release tag {} was not found in tags.".format(from_release))
        print("Available tags: ", tags)
        sys.exit(1)
    return tags[-1]

def get_manifest(session: requests.Session, latest_tag: str):
    """get the plugin manifest file"""

    manifest_headers = {"Accept": "application/vnd.docker.distribution.manifest.v2+json"}
    manifest_url = API_URL + "/" + PLUGIN_URI + "/manifests/" + latest_tag
    manifest_resp = session.get(manifest_url, headers=manifest_headers)
    manifest_resp.raise_for_status()
    return manifest_resp.json()


def main():

    parser = argparse.ArgumentParser(
        description="Builds a new Elastic Log Driver with a custom bindmount",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=USAGE)
    parser.add_argument("bind_target",
                        help="Path on the docker host to bindmount")
    parser.add_argument("--name", default=PLUGIN_URI,
                        help="A custom name for the rebuilt plugin")
    parser.add_argument("--tag", default="",
                        help="A custom tag for the rebuilt plugin")
    parser.add_argument("--from_release", default="",
                        help="Build from an upstream release other than the latest.")
    parser.add_argument("--to_tar", default="",
                        help="Generate a tarball instead of creating a plugin")
    args = parser.parse_args()

    # setup
    session = setup_session()

    print("Fetching tags...")
    latest_tag = get_tag(session, args.from_release)
    print("Using release ", latest_tag)

    print("Fetching plugin manifest...")
    manifest = get_manifest(session, latest_tag)
    config_digest = manifest["config"]["digest"]
    layers = manifest["layers"]
    layer_digest = layers[0]["digest"]

    if len(layers) > 1:
        print("Not expecting a plugin with more than one layer, exiting")
        sys.exit(1)

    # Download the assets from the manifest
    print("Downloading config...")
    cfg_blob_url = API_URL + "/" + PLUGIN_URI + "/blobs/" + config_digest
    config_resp = session.get(cfg_blob_url)
    config_resp.raise_for_status()
    # one half of what we need for building the plugin.
    plugin_config_file = config_resp.json()

    print("Downloading rootfs...")
    # the other half
    rootfs_blob_url = API_URL + "/" + PLUGIN_URI + "/blobs/" + layer_digest
    rootfs_resp = session.get(rootfs_blob_url)
    rootfs_resp.raise_for_status()

    tmpdir = setup_create(plugin_config_file, rootfs_resp.content, args.bind_target)
    new_tag = args.tag if args.tag != "" else latest_tag
    if args.to_tar != "":
        build_archive(tmpdir, args.to_tar, args.name, new_tag)
    else:
        rebuild_plugin(tmpdir, args.name, new_tag)


if __name__ == "__main__":
    try:
        main()
    except requests.HTTPError as err:
        print("Error making HTTP request: ", err)
        sys.exit(1)
