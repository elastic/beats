#!/usr/bin/env python3
import yaml
import os
from dataclasses import dataclass, field

from jinja2 import Template
from pathlib import Path


@dataclass()
class Pipeline:
    """Buildkite Pipeline object"""
    groups: list[str]

    def create_entity(self):
        data = """
steps:
{% for group in pipeline.groups -%}
{{ group.create_entity() }}
{% endfor -%}
"""

        tm = Template(data)
        msg = tm.render(pipeline=self)
        return msg


@dataclass(unsafe_hash=True)
class Group:
    """Buildkite Group object"""

    project: str
    category: str
    steps: list[str]

    def __lt__(self, other):
        return self.project < other.project

    def create_entity(self):
        data = """
{% if group.steps|length > 0 %}
  - group: "{{ group.project }} {{ group.category }}"
    key: "{{ group.project }}-{{ group.category }}"
    steps:
      {% for step in group.steps|sort -%}
        {{ step.create_entity() }}
      {% endfor -%}
{% endif -%}
"""

        tm = Template(data)
        msg = tm.render(group=self)
        return msg


@dataclass(unsafe_hash=True)
class Step:
    """Buildkite Step object"""

    command: str
    name: str
    runner: str
    project: str
    provider: str
    category: str
    label: str = field(init=False)
    comment: str = field(init=False)

    def __post_init__(self):
        self.comment = "/test " + self.project + " " + self.name
        self.label = self.name

    def __lt__(self, other):
        return self.name < other.name

    def create_entity(self):
        data = """
      - label: "{{ stage.project }} {{ stage.name }}"
        command:
          - "{{ stage.command }}"
        notify:
          - github_commit_status:
              context: "{{ stage.project }}: {{ stage.name }}"
        agents:
          provider: "{{ stage.provider }}"
          image: "{{ stage.runner }}"
"""

        tm = Template(data)
        msg = tm.render(stage=self)
        return msg


def is_step_enabled(step: Step, conditions) -> bool:
    # TODO:
    # If PR then
    #   If Changeset

    pull_request = os.getenv('BUILDKITE_PULL_REQUEST')
    if pull_request and pull_request == "false":
        return True

    comment = os.getenv('GITHUB_PR_TRIGGER_COMMENT')
    if comment:
        # the comment should be a subset of the values in .buildkite/pull-requests.json
        # TODO: change /test
        comment_prefix = "buildkite test " + step.project
        # i.e: /test filebeat should run all the mandatory stages
        if step.category == "mandatory" and comment_prefix == comment:
            return True
        # i.e: /test filebeat unitTest
        return comment_prefix + " " + step.name in comment

    labels_env = os.getenv('GITHUB_PR_LABELS')
    if labels_env:
        labels = labels_env.split()
        # i.e: filebeat-unitTest
        if step.project + '-' + step.name in labels:
            return True

    return False


def is_group_enabled(group: Group, conditions) -> bool:
    # TODO:
    # If PR then
    #   If GitHub label matches project name + category (I'm not sure we wanna use this approach since GH comments support it)
    #   If Changeset

    pull_request = os.getenv('BUILDKITE_PULL_REQUEST')
    if pull_request and pull_request == "false":
        return True

    comment = os.getenv('GITHUB_PR_TRIGGER_COMMENT')
    if comment:
        # the comment should be a subset of the values in .buildkite/pull-requests.json
        # TODO: change /test
        comment_prefix = "buildkite test"
        if group.category == "mandatory":
            # i.e: /test filebeat
            return comment_prefix + " " + group.project in comment
        else:
            # i.e: test filebeat extended
            return comment_prefix + " " + group.project + " " + group.category in comment

    return group.category.startswith("mandatory")


def fetch_stage(name: str, stage, project: str, category: str) -> Step:
    """Create a step given the yaml object."""

    # TODO: need to accomodate the provider type.
    # maybe in the buildkite.yml or some dynamic analysis based on the
    # name of the runners.
    return Step(
            category=category,
            command=stage["command"],
            name=name,
            runner=stage["platform"],
            project=project,
            provider="gcp")


def fetch_group(stages, project: str, category: str, conditions) -> Group:
    """Create a group given the yaml object."""

    steps = []

    for stage in stages:
        step = fetch_stage(
                category=category,
                name=stage,
                project=project,
                stage=stages[stage])

        if is_step_enabled(step, conditions):
            steps.append(step)

    return Group(
                project=project,
                category=category,
                steps=steps
            )


# TODO: validate unique stages!

def main() -> None:

    groups = []
    extended_groups = []
    with open(".buildkite/buildkite.yml", "r", encoding="utf8") as file:
        doc = yaml.load(file, yaml.FullLoader)

        for project in doc["projects"]:
            project_file = os.path.join(project, "buildkite.yml")
            if not os.path.isfile(project_file):
                continue
            # TODO: data structure when things run.
            conditions = None
            with open(project_file, "r", encoding="utf8") as file:
                steps = []
                project_obj = yaml.load(file, yaml.FullLoader)

                group = fetch_group(stages=project_obj["stages"]["mandatory"],
                                    project=project,
                                    category="mandatory",
                                    conditions=conditions)

                if is_group_enabled(group, conditions):
                    groups.append(group)

                group = fetch_group(stages=project_obj["stages"]["extended"],
                                    project=project,
                                    category="extended",
                                    conditions=conditions)

                if is_group_enabled(group, conditions):
                    extended_groups.append(group)

    # TODO: improve this merging lists
    all_groups = []
    for group in sorted(groups):
        all_groups.append(group)
    for group in sorted(extended_groups):
        all_groups.append(group)

    # Produce now the pipeline
    print(Pipeline(all_groups).create_entity())


if __name__ == "__main__":

    # pylint: disable=E1120
    main()
