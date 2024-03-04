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
  - group: "{{ group.project }} {{ group.category }}"
    key: "{{ group.project }}-{{ group.category }}"
    steps:
      {% for step in group.steps|sort -%}
        {{ step.create_entity() }}
      {% endfor -%}
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
    # If branch
    # If PR then
    #   If GitHub label matches project name
    #   If GitHub comment
    #   If Changeset
    return True


def is_group_enabled(group: Group, conditions) -> bool:
    # TODO:
    # If branch
    # If PR then
    #   If GitHub label matches project name + category
    #   If GitHub comment
    #   If Changeset
    return True


def fetch_stage(name: str, stage, project: str) -> Step:
    """Create a step given the yaml object."""

    # TODO: need to accomodate the provider type.
    # maybe in the buildkite.yml or some dynamic analysis based on the
    # name of the runners.
    return Step(
            command=stage["command"],
            name=name,
            runner=stage["platform"],
            project=project,
            provider="gcp")

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

                # Given the mandatory list first
                mandatory = project_obj["stages"]["mandatory"]
                for stage in mandatory:
                    step = fetch_stage(
                            name=stage,
                            project=project,
                            stage=mandatory[stage])

                    if is_step_enabled(step, conditions):
                        steps.append(step)

                group = Group(
                            project=project,
                            category="mandatory",
                            steps=steps
                        )

                if is_group_enabled(group, conditions):
                    extended_groups.append(group)

                # Given the extended list if needed
                # TODO: Validate if included
                extended_steps = []

                extended = project_obj["stages"]["extended"]
                for stage in extended:
                    step = fetch_stage(
                            name=stage,
                            project=project,
                            stage=extended[stage])

                    if is_step_enabled(step, conditions):
                        extended_steps.append(step)

                group = Group(
                            project=project,
                            category="extended",
                            steps=extended_steps
                        )

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
