#!/usr/bin/env python3
import platform
import re
import subprocess
import sys
import tarfile

PATH = 'x-pack/agentbeat/build/distributions'
PLATFORMS = {
                'windows': {
                    'amd64': 'x86_64',
                },
                'linux': {
                    'x86_64': 'x86_64',
                    'aarch64': 'arm64',
                },
                'darwin': {
                    'x86_64': 'x86_64',
                    'arm64': 'aarch64',
                }
            }


class Archive:
    def __init__(self, os, arch, ext):
            self.os = os
            self.arch = arch
            self.ext = ext


def log(msg):
    sys.stdout.write(f'{msg}\n')
    sys.stdout.flush()


def log_err(msg):
    sys.stderr.write(f'{msg}\n')
    sys.stderr.flush()


def get_archive_params() -> Archive:
    system = platform.system().lower()
    machine = platform.machine().lower()
    arch = PLATFORMS.get(system, {}).get(machine)
    ext = get_artifact_extension(system)

    return Archive(system, arch, ext)


def get_artifact_extension(system) -> str:
    if system == 'windows':
        return 'zip'
    else:
        return 'tar.gz'


def get_artifact_pattern(archive_obj) -> str:
    return f'{PATH}/agentbeat-*-{archive_obj.os}-{archive_obj.arch}.{archive_obj.ext}'


def download_agentbeat(archive_obj) -> str:
    pattern = get_artifact_pattern(archive_obj)
    log('--- Downloading Agentbeat artifact by pattern: ' + pattern)
    try:
        subprocess.run(
            ['buildkite-agent', 'artifact', 'download', pattern, '.',
             '--step', 'agentbeat-package-linux'],
            check=True, stdout=sys.stdout, stderr=sys.stderr, text=True)

    except subprocess.CalledProcessError:
        exit(1)

    return get_full_filename()


def get_full_filename() -> str:
    try:
        out = subprocess.run(
            ['ls', '-p', PATH],
            check=True, capture_output=True, text=True)
        return out.stdout.strip()
    except subprocess.CalledProcessError:
        exit(1)


def extract_agentbeat(filename):
    filepath = PATH + '/' + filename
    log('Extracting Agentbeat artifact: ' + filepath)

    if filepath.endswith('.zip'):
        unzip_agentbeat(filepath)
    else:
        untar_agentbeat(filepath)


def unzip_agentbeat(filepath):
    try:
        subprocess.run(
            ['unzip', '-qq', filepath],
            check=True, stdout=sys.stdout, stderr=sys.stderr, text=True)
    except subprocess.CalledProcessError as e:
        log_err(e)
        exit(1)


def untar_agentbeat(filepath):
    try:
        subprocess.run(
            ['tar', '-xvf', filepath],
            check=True, stdout=sys.stdout, stderr=sys.stderr, text=True)
    except subprocess.CalledProcessError as e:
        log_err(e)
        exit(1)


def get_path_to_executable(filepath) -> str:
    pattern = r'(.*)(?=\.zip|.tar\.gz)'
    match = re.match(pattern, filepath)
    if match:
        path = f'../../{match.group(1)}/agentbeat'
        return path
    else:
        log_err('No agentbeat executable found')
        exit(1)

archive_params = get_archive_params()
archive = download_agentbeat(archive_params)
extract_agentbeat(archive)
log(get_path_to_executable(archive))
