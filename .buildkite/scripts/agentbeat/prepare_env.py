#!/usr/bin/env python3

import os
import platform
import subprocess


def get_os() -> str:
    return platform.system()


def get_arch() -> str:
    return platform.machine()


def get_cwd() -> str:
    return os.getcwd()


def download_agentbeat_artifact(os, arch):
    pattern = "x-pack/agentbeat/build/distributions/agentbeat-9.0.0-SNAPSHOT-linux-x86_64.tar.gz"
    # pattern = "x-pack/agentbeat/build/distributions/**"
    # command = f"buildkite-agent artifact download \"{pattern}\" . --step 'agentbeat-package-linux'"

    try:
        print("--- Downloading agentbeat artifact")
        result = subprocess.run(
            ["buildkite-agent", "artifact", "download", pattern, ".",
             "--build", "01924d2b-b061-45ae-a106-e885584ff26f",
             "--step", "agentbeat-package-linux"],
            check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        print(result.stdout.decode())
    except subprocess.CalledProcessError as e:
        print("--- Error occurred while downloading agentbeat\n" + e.stderr)
        exit(1)


def install_synthetics():
    try:
        print("--- Installing @elastic/synthetics")
        subprocess.run(
            ["npm install -g @elastic/synthetics"],
            check=True
        )
    except subprocess.CalledProcessError:
        print("Failed to install @elastic/synthetics")
        exit(1)


# print("--- OS: " + get_os())
#
# print("--- ARCH: " + get_arch())
#
# print("--- CWD: " + get_cwd())

download_agentbeat_artifact(get_os(), get_arch())
# install_synthetics()
