#!/usr/bin/env python3

'''
This script is used by schedule-type pipelines
to automate triggering other pipelines (e.g. Iron Bank validation)
against release branches

Excepts a (comma separated) env var PIPELINES_TO_TRIGGER.
An optional EXCLUDE_BRANCHES (comma separated) env var can also be supplied to skip specific branches (e.g. EXCLUDE_BRANCHES="main")

For background info see:
https://elasticco.atlassian.net/browse/ENGPRD-318 /
https://github.com/elastic/ingest-dev/issues/2664
'''

import json
import os
import sys
import time
import typing
import urllib.request
from ruamel.yaml import YAML


ACTIVE_BRANCHES_URL = "https://storage.googleapis.com/artifacts-api/snapshots/branches.json"


class InputError(Exception):
    """ Exception raised for input errors """


class UrlOpenError(Exception):
    """ Exception raised when hitting errors retrieving content from a URL """


def fail_with_error(msg):
    print(f"""^^^ +++
Error: [{msg}].
Exiting now.
    """)
    exit(1)


def parse_csv_env_var(env_var_name: str, is_valid=False) -> typing.List:
    if is_valid and env_var_name not in os.environ.keys():
        fail_with_error(msg=f'Required environment variable [{env_var_name}] is missing.')

    env_var = os.getenv(env_var_name, "")

    if is_valid and env_var.strip() == "":
        fail_with_error(msg=f'Required environment variable [{env_var_name}] is empty.')
    return env_var.split(",")


def get_json_with_retries(uri, retries=3, delay=5) -> typing.Dict:
    for _ in range(retries):
        try:
            with urllib.request.urlopen(uri) as response:
                data = response.read().decode('utf-8')
                return json.loads(data)
        except UrlOpenError as e:
            print(f"Error: [{e}] when downloading from [{uri}]")
            print(f"Retrying in {delay} seconds ...")
            time.sleep(delay)
        except json.JSONDecodeError as e:
            fail_with_error(f"Error [{e}] when deserialing JSON from [{uri}]")
    fail_with_error(f"Failed to retrieve JSON content from [{uri}] after [{retries}] retries")
    return {}  # for IDE typing checks


def get_release_branches() -> typing.List[str]:
    resp = get_json_with_retries(uri=ACTIVE_BRANCHES_URL)
    try:
        release_branches = [branch for branch in resp["branches"]]
    except KeyError:
        fail_with_error(f'''Didn't find the excepted structure ["branches"] in the response [{resp}] from [{ACTIVE_BRANCHES_URL}]''')

    return release_branches


def generate_pipeline(pipelines_to_trigger: typing.List[str], branches: typing.List[str]):
    generated_pipeline = {"steps": []}

    for pipeline in pipelines_to_trigger:
        for branch in branches:
            trigger = {
                "trigger": pipeline,
                "label": f":testexecute: Triggering {pipeline} / {branch}",
                "build": {
                    "branch": branch,
                    "message": f":testexecute: Scheduled build for {branch}"
                }
            }
            generated_pipeline["steps"].append(trigger)

    return generated_pipeline


if __name__ == '__main__':
    pipelines_to_trigger = parse_csv_env_var(env_var_name="PIPELINES_TO_TRIGGER", is_valid=True)
    release_branches = get_release_branches()
    exclude_branches = parse_csv_env_var(env_var_name="EXCLUDE_BRANCHES")

    target_branches = sorted(list(set(release_branches).difference(exclude_branches)))
    if len(target_branches) == 0 or target_branches[0].isspace():
        fail_with_error(f"Calculated target branches were empty! You passed EXCLUDE_BRANCHES={exclude_branches} and release branches are {release_branches} the difference of which results in {target_branches}.")

    pipeline = generate_pipeline(pipelines_to_trigger, branches=target_branches)
    print('# yaml-language-server: $schema=https://raw.githubusercontent.com/buildkite/pipeline-schema/main/schema.json')
    YAML().dump(pipeline, sys.stdout)
