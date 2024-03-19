#!/usr/bin/env python3
from typing import Any
from ruamel.yaml import YAML
import os
import subprocess
import fnmatch
import sys


class Agent:
    """Buildkite Agent object"""

    def __init__(self, image: str, provider: str):
        self.image: str = image
        self.provider: str = provider

    def create_entity(self):
        raise NotImplementedError("Not implemented yet")


class AWSAgent(Agent):
    """AWS Agent object"""

    def __init__(self, image: str, instance_type: str = None):
        super().__init__(image, "aws")
        if instance_type is None:
            self.instance_type: str = "t4g.large"
        else:
            self.instance_type = instance_type

    def create_entity(self) -> dict[str, str]:
        return {
            "provider": self.provider,
            "imagePrefix": self.image,
            "instanceType": self.instance_type,
        }


class GCPAgent(Agent):
    """GCP Agent object"""

    def __init__(self, image: str):
        super().__init__(image, "gcp")

    def create_entity(self) -> dict[str, str]:
        return {
            "provider": self.provider,
            "image": self.image,
        }


class OrkaAgent(Agent):
    """Orka Agent object"""

    def __init__(self, image: str):
        super().__init__(image, "orka")

    def create_entity(self) -> dict[str, str]:
        return {
            "provider": self.provider,
            "imagePrefix": self.image,
        }


class Step:
    """Buildkite Step object"""

    def __init__(
        self, name: str, command: str, project: str, category: str, agent: Agent
    ):
        self.command: str = command
        self.agent: Agent = agent
        self.name: str = name
        self.project: str = project
        self.category: str = category
        self.comment = "/test " + self.project + " " + self.name
        self.label = self.name

    def __lt__(self, other):
        return self.name < other.name

    def create_entity(self) -> dict[str, Any]:
        data = {
            "label": f"{self.project} {self.name}",
            "command": [self.command],
            "notify": [
                {
                    "github_commit_status": {
                        "context": f"{self.project}: {self.name}",
                    }
                }
            ],
            "agents": self.agent.create_entity(),
            "artifact_paths": [
                f"{self.project}/build/*.xml",
                f"{self.project}/build/*.json"
            ],
        }
        return data


class Group:
    """Buildkite Group object"""

    def __init__(self, project: str, category: str, steps: list[Step]):
        self.project: str = project
        self.category: str = category
        self.steps: list[Step] = steps

    def __lt__(self, other):
        return self.project < other.project

    def create_entity(self) -> dict[str, Any]:
        if len(self.steps) == 0:
            return {}

        data = {
            "group": f"{self.project} {self.category}",
            "key": f"{self.project}-{self.category}",
            "steps": [step.create_entity() for step in self.steps],
        }

        return data


class GitHelper:
    def __init__(self):
        self.files: list[str] = []

    def get_pr_changeset(self) -> list[str]:
        base_branch = os.getenv("BUILDKITE_PULL_REQUEST_BASE_BRANCH", "main")
        diff_command = ["git", "diff", "--name-only", "{}...HEAD".format(base_branch)]
        result = subprocess.run(diff_command, stdout=subprocess.PIPE)
        if result.returncode == 0:
            self.files = result.stdout.decode().splitlines()
        else:
            print(f"Detecting changed files failed, exiting [{result.returncode}]")
            exit(result.returncode)
        return self.files


class BuildkitePipeline:
    """Buildkite Pipeline object"""

    def __init__(self, groups: list[Group] = None):
        if groups is None:
            groups = []
        self.groups: list[Group] = groups

    def create_entity(self):
        data = {"steps": [group.create_entity() for group in self.groups]}
        return data


def is_pr() -> bool:
    return os.getenv("BUILDKITE_PULL_REQUEST") != "false"


def group_comment(group: Group) -> bool:
    comment = os.getenv("GITHUB_PR_TRIGGER_COMMENT")
    if comment:
        # the comment should be a subset of the values
        # in .buildkite/pull-requests.json
        # TODO: change /test
        comment_prefix = "buildkite test"
        if group.category == "mandatory":
            # i.e: /test filebeat
            return comment_prefix + " " + group.project in comment
        else:
            # i.e: test filebeat extended
            return (
                comment_prefix + " " + group.project + " " + group.category in comment
            )


def filter_files_by_glob(files, patterns: list[str]):
    for pattern in patterns:
        # TODO: Support glob extended patterns: ^ and etc.
        # Now it supports only linux glob syntax
        if fnmatch.filter(files, pattern):
            return True
    return False


def is_in_pr_changeset(
    project_changeset_filters: list[str], changeset: list[str]
) -> bool:
    return filter_files_by_glob(changeset, project_changeset_filters)


def is_group_enabled(
    group: Group, changeset_filters: list[str], changeset: list[str]
) -> bool:
    if not is_pr():
        return True

    if (
        is_pr()
        and is_in_pr_changeset(changeset_filters, changeset)
        and group.category.startswith("mandatory")
    ):
        return True

    return group_comment(group)


def fetch_stage(name: str, stage, project: str, category: str) -> Step:
    """Create a step given the yaml object."""

    agent: Agent = None
    if ("provider" not in stage) or stage["provider"] == "gcp":
        agent = GCPAgent(image=stage["platform"])
    elif stage["provider"] == "aws":
        agent = AWSAgent(
            image=stage["platform"],
        )
    elif stage["provider"] == "orka":
        agent = OrkaAgent(image=stage["platform"])

    return Step(
        category=category,
        command=stage["command"],
        name=name,
        agent=agent,
        project=project,
    )


def fetch_group(stages, project: str, category: str) -> Group:
    """Create a group given the yaml object."""

    steps = []

    for stage in stages:
        steps.append(
            fetch_stage(
                category=category, name=stage, project=project, stage=stages[stage]
            )
        )

    return Group(project=project, category=category, steps=steps)


def fetch_pr_pipeline(yaml: YAML) -> list[Group]:
    git_helper = GitHelper()
    changeset = git_helper.get_pr_changeset()
    groups: list[Group] = []
    doc = pipeline_loader(yaml)
    for project in doc["projects"]:
        project_file = os.path.join(project, "buildkite.yml")
        if not os.path.isfile(project_file):
            continue
        project_obj = project_loader(yaml, project_file)
        group = fetch_group(
            stages=project_obj["stages"]["mandatory"],
            project=project,
            category="mandatory",
        )

        if is_group_enabled(group, project_obj["when"]["changeset"], changeset):
            groups.append(group)

        group = fetch_group(
            stages=project_obj["stages"]["extended"],
            project=project,
            category="extended",
        )

        if is_group_enabled(group, project_obj["when"]["changeset"], changeset):
            groups.append(group)

    # TODO: improve this merging lists
    all_groups = []
    for group in groups:
        all_groups.append(group)

    return all_groups


class PRComment:
    command: str
    group: str
    project: str
    step: str

    def __init__(self, comment: str):
        words = comment.split()
        self.command = words.pop(0) if words else ""
        self.project = words.pop(0) if words else ""
        self.group = words.pop(0) if words else ""
        self.step = words.pop(0) if words else ""


# A comment like "/test filebeat extended"
# Returns a group of steps corresponding to the comment
def fetch_pr_comment_group_pipeline(comment: PRComment, yaml: YAML) -> list[Group]:
    groups = []
    doc = pipeline_loader(yaml)
    if comment.project in doc["projects"]:
        project_file = os.path.join(comment.project, "buildkite.yml")
        if not os.path.isfile(project_file):
            raise FileNotFoundError(
                "buildkite.yml not found in: " + "{}".format(comment.project)
            )
        project_obj = project_loader(yaml, project_file)
        if not project_obj["stages"][comment.group]:
            raise ValueError(
                "Group not found in {} buildkite.yml: {}".format(
                    comment.project, comment.group
                )
            )

        group = fetch_group(
            stages=project_obj["stages"][comment.group],
            project=comment.project,
            category="mandatory",
        )
        groups.append(group)

    return groups


# A comment like "/test filebeat extended unitTest-macos"
def fetch_pr_comment_step_pipeline(comment: PRComment, yaml: YAML) -> list[Group]:
    groups = []
    doc = pipeline_loader(yaml)
    if comment.project in doc["projects"]:
        project_file = os.path.join(comment.project, "buildkite.yml")
        if not os.path.isfile(project_file):
            raise FileNotFoundError(
                "buildkite.yml not found in: " + "{}".format(comment.project)
            )
        project_obj = project_loader(yaml, project_file)
        if not project_obj["stages"][comment.group]:
            raise ValueError(
                "Group not found in {} buildkite.yml: {}".format(
                    comment.project, comment.group
                )
            )
        group = fetch_group(
            stages=project_obj["stages"][comment.group],
            project=comment.project,
            category="mandatory",
        )

        filtered_steps = list(
            filter(lambda step: step.name == comment.step, group.steps)
        )

        if not filtered_steps:
            raise ValueError(
                "Step {} not found in {} buildkite.yml".format(
                    comment.step, comment.project
                )
            )
        group.steps = filtered_steps
        groups.append(group)

    return groups


def pr_comment_pipeline(pr_comment: PRComment, yaml: YAML) -> list[Group]:

    if pr_comment.command == "/test":

        # A comment like "/test" for a PR
        # We rerun the PR pipeline
        if not pr_comment.group:
            return fetch_pr_pipeline(yaml)

        # A comment like "/test filebeat"
        # We don't know what group to run hence raise an error
        if pr_comment.project and not pr_comment.group:
            raise ValueError(
                "Specify group or/and step for {}".format(pr_comment.project)
            )

        # A comment like "/test filebeat extended"
        # We rerun the filebeat extended pipeline for the PR
        if pr_comment.group and not pr_comment.step:
            return fetch_pr_comment_group_pipeline(pr_comment, yaml)

        # A comment like "/test filebeat extended unitTest-macos"
        if pr_comment.step:
            return fetch_pr_comment_step_pipeline(pr_comment, yaml)


# TODO: validate unique stages!
def main() -> None:
    yaml = YAML(typ="safe")
    all_groups = []
    if is_pr() and not os.getenv("GITHUB_PR_TRIGGER_COMMENT"):
        all_groups = fetch_pr_pipeline(yaml)

    if is_pr() and os.getenv("GITHUB_PR_TRIGGER_COMMENT"):
        print(
            "GITHUB_PR_TRIGGER_COMMENT: {}".format(
                os.getenv("GITHUB_PR_TRIGGER_COMMENT")
            )
        )
        comment = PRComment(os.getenv("GITHUB_PR_TRIGGER_COMMENT"))
        all_groups = pr_comment_pipeline(comment, yaml)

    # Produce the dynamic pipeline
    print(
        "# yaml-language-server: $schema=https://raw.githubusercontent.com/buildkite/pipeline-schema/main/schema.json"
    )
    yaml.dump(BuildkitePipeline(all_groups).create_entity(), sys.stdout)


def pipeline_loader(yaml: YAML = YAML(typ="safe")):
    with open(".buildkite/buildkite.yml", "r", encoding="utf8") as file:
        return yaml.load(file)


def project_loader(yaml: YAML = YAML(typ="safe"), project_file: str = ""):
    with open(project_file, "r", encoding="utf8") as project_fp:
        return yaml.load(project_fp)


if __name__ == "__main__":

    # pylint: disable=E1120
    main()
