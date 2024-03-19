#!/usr/bin/env python3
from typing import Any
import yaml
import os
from dataclasses import dataclass, field
import subprocess
import fnmatch


@dataclass(unsafe_hash=True)
class Agent:
    """Buildkite Agent object"""
    image: str

    def create_entity(self):
        raise NotImplementedError("Not implemented yet")


@dataclass(unsafe_hash=True)
class AWSAgent(Agent):
    """AWS Agent object"""
    image: str

    def create_entity(self) -> dict[str, dict[str, str]]:
        return {
            "agents": {
                "provider": "aws",
                "imagePrefix": self.image,
                "instanceType": "t4g.large",
            }
        }


@dataclass(unsafe_hash=True)
class GCPAgent(Agent):
    """GCP Agent object"""
    image: str

    def create_entity(self) -> dict[str, dict[str, str]]:
        return {
            "agents": {
                "provider": "gcp",
                "image": self.image,
            }
        }


@dataclass(unsafe_hash=True)
class OrkaAgent(Agent):
    """Orka Agent object"""
    image: str

    def create_entity(self) -> dict[str, dict[str, str]]:
        return {
            "agents": {
                "provider": "orka",
                "imagePrefix": self.image,
            }
        }


@dataclass(unsafe_hash=True)
class Step:
    """Buildkite Step object"""

    command: str
    agent: Agent
    name: str
    project: str
    category: str
    label: str = field(init=False)
    comment: str = field(init=False)

    def __post_init__(self):
        self.comment = "/test " + self.project + " " + self.name
        self.label = self.name

    def __lt__(self, other):
        return self.name < other.name

    def create_entity(self) -> dict[str, Any]:
        data = {
            "label": f"{self.project} {self.name}",
            "command": [self.command],
            "notify": [{
                "github_commit_status": {
                    "context": f"{self.project}: {self.name}",
                }
            }],
        }
        data.update(self.agent.create_entity())
        return data


@dataclass(unsafe_hash=True)
class Group:
    """Buildkite Group object"""

    project: str
    category: str
    steps: list[Step]

    def __lt__(self, other):
        return self.project < other.project

    def create_entity(self) -> dict[str, Any] | None:
        if len(self.steps) == 0:
            return

        data = {
            "group": f"{self.project} {self.category}",
            "key": f"{self.project}-{self.category}",
            "steps": [step.create_entity() for step in sorted(self.steps)],
        }

        return data


@dataclass()
class Pipeline:
    """Buildkite Pipeline object"""
    groups: list[Group]

    def create_entity(self):
        data = {"steps": [group.create_entity() for group in self.groups]}
        return data


def is_pr() -> bool:
    return os.getenv('BUILDKITE_PULL_REQUEST') != "false"


def group_comment(group: Group) -> bool:
    comment = os.getenv('GITHUB_PR_TRIGGER_COMMENT')
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
                comment_prefix + " " + group.project + " " + group.category
                in comment
            )


changed_files = None


def get_pr_changeset():
    global changed_files
    if not changed_files:
        base_branch = os.getenv('BUILDKITE_PULL_REQUEST_BASE_BRANCH')
        diff_command = [
            "git", "diff", "--name-only", "{}...HEAD".format(base_branch)
        ]
        result = subprocess.run(diff_command, stdout=subprocess.PIPE)
        changed_files = result.stdout.decode().splitlines()
        print("Changed files: {}".format(changed_files))
    return changed_files


def filter_files_by_glob(files, patterns: list[str]):
    for pattern in patterns:
        # TODO: Support glob extended patterns: ^ and etc.
        # Now it supports only linux glob syntax
        if fnmatch.filter(files, pattern):
            return True
    return False


def is_in_pr_changeset(project_changeset_filters: list[str]) -> bool:
    changeset = get_pr_changeset()
    return filter_files_by_glob(changeset, project_changeset_filters)


def is_group_enabled(group: Group, changeset_filters: list[str]) -> bool:
    if not is_pr():
        return True

    if is_pr() and is_in_pr_changeset(changeset_filters) and \
            group.category.startswith("mandatory"):
        return True

    return group_comment(group)


def fetch_stage(name: str, stage, project: str, category: str) -> Step:
    """Create a step given the yaml object."""

    agent: Agent = None
    if ("provider" not in stage) or stage["provider"] == "gcp":
        agent = GCPAgent(image=stage["platform"])
    elif stage["provider"] == "aws":
        agent = AWSAgent(image=stage["platform"])
    elif stage["provider"] == "orka":
        agent = OrkaAgent(image=stage["platform"])

    return Step(
            category=category,
            command=stage["command"],
            name=name,
            agent=agent,
            project=project)


def fetch_group(stages, project: str, category: str) -> Group:
    """Create a group given the yaml object."""

    steps = []

    for stage in stages:
        steps.append(fetch_stage(
                category=category,
                name=stage,
                project=project,
                stage=stages[stage]))

    return Group(
                project=project,
                category=category,
                steps=steps)


def fetch_pr_pipeline() -> list[Group]:
    groups = []
    extended_groups = []
    with open(".buildkite/buildkite.yml", "r", encoding="utf8") as file:
        doc = yaml.load(file, yaml.FullLoader)

        for project in doc["projects"]:
            project_file = os.path.join(project, "buildkite.yml")
            if not os.path.isfile(project_file):
                continue
            with open(project_file, "r", encoding="utf8") as file:
                project_obj = yaml.load(file, yaml.FullLoader)

                group = fetch_group(stages=project_obj["stages"]["mandatory"],
                                    project=project,
                                    category="mandatory")

                if is_group_enabled(group, project_obj["when"]["changeset"]):
                    groups.append(group)

                group = fetch_group(stages=project_obj["stages"]["extended"],
                                    project=project,
                                    category="extended")

                if is_group_enabled(group, project_obj["when"]["changeset"]):
                    extended_groups.append(group)

    # TODO: improve this merging lists
    all_groups = []
    for group in sorted(groups):
        all_groups.append(group)
    for group in sorted(extended_groups):
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
def fetch_pr_comment_group_pipeline(comment: PRComment) -> list[Group]:
    groups = []
    with open(".buildkite/buildkite.yml", "r", encoding="utf8") as file:
        doc = yaml.load(file, yaml.FullLoader)
        if comment.project in doc["projects"]:
            project_file = os.path.join(comment.project, "buildkite.yml")

            if not os.path.isfile(project_file):
                raise FileNotFoundError("buildkite.yml not found in: " +
                                        "{}".format(comment.project))
            with open(project_file, "r", encoding="utf8") as file:
                project_obj = yaml.load(file, yaml.FullLoader)

                if not project_obj["stages"][comment.group]:
                    raise ValueError("Group not found in {} buildkike.yml: {}"
                                     .format(comment.project, comment.group))

                group = fetch_group(
                    stages=project_obj["stages"][comment.group],
                    project=comment.project,
                    category="mandatory"
                )
                groups.append(group)

    return groups


# A comment like "/test filebeat extended unitTest-macos"
def fetch_pr_comment_step_pipeline(comment: PRComment) -> list[Group]:
    groups = []
    with open(".buildkite/buildkite.yml", "r", encoding="utf8") as file:
        doc = yaml.load(file, yaml.FullLoader)
        if comment.project in doc["projects"]:
            project_file = os.path.join(comment.project, "buildkite.yml")

            if not os.path.isfile(project_file):
                raise FileNotFoundError("buildkite.yml not found in: " +
                                        "{}".format(comment.project))
            with open(project_file, "r", encoding="utf8") as file:
                project_obj = yaml.load(file, yaml.FullLoader)

                if not project_obj["stages"][comment.group]:
                    raise ValueError("Group not found in {} buildkike.yml: {}"
                                     .format(comment.project, comment.group))

                group = fetch_group(
                    stages=project_obj["stages"][comment.group],
                    project=comment.project,
                    category="mandatory"
                )

                filtered_steps = list(filter(
                    lambda step: step.name == comment.step,
                    group.steps
                ))

                if not filtered_steps:
                    raise ValueError("Step {} not found in {} buildkike.yml"
                                     .format(comment.step, comment.project))
                group.steps = filtered_steps
                groups.append(group)

        return groups


def pr_comment_pipeline(pr_comment: PRComment) -> list[Group]:

    if pr_comment.command == "/test":

        # A comment like "/test" for a PR
        # We rerun the PR pipeline
        if not pr_comment.group:
            return fetch_pr_pipeline()

        # A comment like "/test filebeat"
        # We don't know what group to run hence raise an error
        if pr_comment.project and not pr_comment.group:
            raise ValueError(
                "Specify group or/and step for {}".format(pr_comment.project)
            )

        # A comment like "/test filebeat extended"
        # We rerun the filebeat extended pipeline for the PR
        if pr_comment.group and not pr_comment.step:
            return fetch_pr_comment_group_pipeline(pr_comment)

        # A comment like "/test filebeat extended unitTest-macos"
        if pr_comment.step:
            return fetch_pr_comment_step_pipeline(pr_comment)


# TODO: validate unique stages!
def main() -> None:
    all_groups = []
    if is_pr() and not os.getenv('GITHUB_PR_TRIGGER_COMMENT'):
        all_groups = fetch_pr_pipeline()

    if is_pr() and os.getenv('GITHUB_PR_TRIGGER_COMMENT'):
        print("GITHUB_PR_TRIGGER_COMMENT: {}".format(
            os.getenv('GITHUB_PR_TRIGGER_COMMENT')))
        comment = PRComment(os.getenv('GITHUB_PR_TRIGGER_COMMENT'))
        all_groups = pr_comment_pipeline(comment)

    # Produce now the pipeline
    print(yaml.dump(Pipeline(all_groups).create_entity()))


if __name__ == "__main__":

    # pylint: disable=E1120
    main()
