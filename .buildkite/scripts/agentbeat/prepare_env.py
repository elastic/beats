#!/usr/bin/env python3

import platform
import subprocess
import sys


def get_os() -> str:
    return platform.system().lower()


def get_arch() -> str:
    arch = platform.machine().lower()

    if arch == "amd64":
        return "x86_64"
    else:
        return arch


def get_artifact_extension(agent_os) -> str:
    if agent_os == "windows":
        return "zip"
    else:
        return "tar.gz"


def download_agentbeat_artifact(agent_os, agent_arch):
    print(" ")
    extension = get_artifact_extension(agent_os)
    pattern = f"x-pack/agentbeat/build/distributions/agentbeat-*-{agent_os}-{agent_arch}.{extension}"

    try:
        subprocess.run(
            ["buildkite-agent", "artifact", "download", pattern, ".",
             "--build", "01924d2b-b061-45ae-a106-e885584ff26f",
             "--step", "agentbeat-package-linux"],
            check=True, stdout=sys.stdout, stderr=sys.stderr, text=True)
    except subprocess.CalledProcessError:
        exit(1)


def unzip_agentbeat():
    print("todo unzip")
    try:
        subprocess.run(
            ["unzip"],
            check=True, stdout=sys.stdout, stderr=sys.stderr, text=True)
    except subprocess.CalledProcessError:
        exit(1)


def install_synthetics():
    print("--- Installing @elastic/synthetics")

    try:
        subprocess.run(
            ["npm install -g @elastic/synthetics"],
            check=True
        )
    except subprocess.CalledProcessError:
        print("Failed to install @elastic/synthetics")
        exit(1)

print("--- OS Data: " + get_os() + " " + get_arch())
download_agentbeat_artifact(get_os(), get_arch())
# install_synthetics()
