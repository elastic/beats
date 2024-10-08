#!/usr/bin/env python3

import platform
import subprocess
import sys
import tarfile
import os
import re

PATH = 'x-pack/agentbeat/build/distributions'


def log(msg):
    sys.stdout.write(f'{msg}\n')
    sys.stdout.flush()

def log_err(msg):
    sys.stderr.write(f'{msg}\n')
    sys.stderr.flush()

def get_os() -> str:
    return platform.system().lower()


def get_arch() -> str:
    arch = platform.machine().lower()

    if arch == 'amd64':
        return 'x86_64'
    else:
        return arch


def get_artifact_extension(agent_os) -> str:
    if agent_os == 'windows':
        return 'zip'
    else:
        return 'tar.gz'


def get_artifact_pattern() -> str:
    agent_os = get_os()
    agent_arch = get_arch()
    extension = get_artifact_extension(agent_os)
    return f'{PATH}/agentbeat-*-{agent_os}-{agent_arch}.{extension}'


def download_agentbeat(pattern, path) -> str:
    log('--- Downloading agentbeat')
    try:
        subprocess.run(
            ['buildkite-agent', 'artifact', 'download', pattern, '.',
             '--build', '01924d2b-b061-45ae-a106-e885584ff26f',
             '--step', 'agentbeat-package-linux'],
            check=True, stdout=sys.stdout, stderr=sys.stderr, text=True)
    except subprocess.CalledProcessError:
        exit(1)

    return get_filename(path)


def get_filename(path) -> str:
    try:
        out = subprocess.run(
            ['ls', '-p', path],
            check=True, capture_output=True, text=True)
        return out.stdout.strip()
    except subprocess.CalledProcessError:
        exit(1)


def extract_agentbeat(filename):
    log('~~~ Extracting agentbeat')
    filepath = PATH + '/' + filename

    if filepath.endswith('.zip'):
        unzip_agentbeat(filepath)
    else:
        untar_agentbeat(filepath)
    log('Successfully extracted agentbeat')


def unzip_agentbeat(filepath):
    try:
        subprocess.run(
            ['unzip', filepath],
            check=True, stdout=sys.stdout, stderr=sys.stderr, text=True)
    except subprocess.CalledProcessError as e:
        log_err(e)
        exit(1)


def untar_agentbeat(filepath):
    try:
        with tarfile.open(filepath, 'r:gz') as tar:
            tar.list()
            tar.extractall()
    except Exception as e:
        log_err(e)
        exit(1)


def add_to_path(filepath):
    pattern = r'(.*)(?=\.zip|.tar\.gz)'
    match = re.match(pattern, filepath)
    if match:
        path = f'../build/distributions/{match.group(1)}/agentbeat'
        log("--- PATH: " + str(path))
        os.environ['AGENTBEAT_PATH'] = str(path)
    else:
        log_err("No agentbeat executable found")
        exit(1)


def install_synthetics():
    log('--- Installing @elastic/synthetics')

    try:
        subprocess.run(
            ['npm', 'install', '-g', '@elastic/synthetics'],
            check=True
        )
    except subprocess.CalledProcessError:
        log_err('Failed to install @elastic/synthetics')
        exit(1)


artifact_pattern = get_artifact_pattern()
archive = download_agentbeat(artifact_pattern, PATH)
extract_agentbeat(archive)
add_to_path(archive)
install_synthetics()
