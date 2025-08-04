import argparse
from collections import OrderedDict
from functools import lru_cache
import os
import re
import requests

import yaml


def document_fields(output, section, sections, path, beat):
    if "skipdocs" in section:
        return
    if "anchor" in section:
        output.write('---\n')
        output.write('mapped_pages:\n')
        output.write(
            '  - https://www.elastic.co/guide/en/beats/{}/current/exported-fields-{}.html\n'.format(beat, section["anchor"]))
        # Build a docs-builder applies_to directive
        applies_to = get_applies_to(section)
        if len(applies_to) > 0:
            output.write('applies_to:\n')
            output.write('  stack: ')
            output.write(', '.join(applies_to))
            output.write('\n')
        output.write('---\n\n')
        output.write('% This file is generated! See scripts/generate_fields_docs.py\n\n')

    # Intermediate level titles
    # if ("description" in section and "prefix" not in section and
    #         "anchor" not in section):
    #     output.write("[float]\n")

    if "description" in section:
        if section["description"] is None:
            section["description"] = "None"
        if "anchor" in section and section["name"] == "ECS":
            output.write("# {} fields [exported-fields-ecs]\n\n".format(section["name"]))
            output.write("""This section defines Elastic Common Schema (ECS) fields—a common set of fields
to be used when storing event data in {{{{es}}}}.

This is an exhaustive list, and fields listed here are not necessarily used by {beatname_uc}.
The goal of ECS is to enable and encourage users of {{{{es}}}} to normalize their event data,
so that they can better analyze, visualize, and correlate the data represented in their events.

See the [ECS reference](ecs://reference/index.md) for more information.
""".format_map({"beatname_uc": beat.title()}))
        elif "anchor" in section:
            output.write("# {} fields [exported-fields-{}]\n\n".format(section["title"], section["anchor"]))
            output.write("{}\n\n".format(section["description"].strip()))
        else:
            output.write("## {} [_{}]\n\n".format(section["name"], section["name"]))

            # Build a docs-builder applies_to directive
            applies_to = get_applies_to(section)
            if len(applies_to) > 0:
                output.write("```{applies_to}\nstack: ")
                output.write(", ".join(applies_to))
                output.write("\n```\n\n")

            output.write("{}\n\n".format(section["description"].strip()))

    if "fields" not in section or not section["fields"]:
        return

    for field in section["fields"]:
        if "description" in field and field["description"] is None:
            field["description"] = "None"
        # Skip entries which do not define a name
        if "name" not in field:
            continue

        if path == "":
            newpath = field["name"]
        else:
            newpath = path + "." + field["name"]

        if "type" in field and field["type"] == "group":
            document_fields(output, field, sections, newpath, beat)
        else:
            document_field(output, field, newpath)


def document_field(output, field, field_path):

    if "field_path" not in field:
        field["field_path"] = field_path

    output.write("**`{}`**".format(field["field_path"]))

    # Build a docs-builder applies_to directive
    applies_to = get_applies_to(field)
    if len(applies_to) > 0:
        output.write(" {applies_to}`stack: ")
        output.write(", ".join(applies_to))
        output.write("`")

    output.write("\n:   ")

    if "description" in field and field["description"] is not None and len(field["description"].strip()) > 0:
        output.write("{}".format(" ".join(x for x in field["description"].split("\n") if x)).strip()+"\n\n")
    if "type" in field:
        if "description" in field:
            output.write("    ")
        output.write("type: {}\n\n".format(field["type"]))
    if "example" in field:
        output.write("    example: {}\n\n".format(field["example"]))
    if "format" in field:
        output.write("    format: {}\n\n".format(field["format"]))
    if "required" in field:
        output.write("    required: {}\n\n".format(field["required"]))
    if "path" in field:
        output.write("    alias to: {}\n\n".format(field["path"]))

    # For Apm-Server docs only
    # Assign an ECS badge for ECS fields
    if beat_title == "Apm-Server":
        if field_path in ecs_fields():
            output.write("{yes-icon} {ecs-ref}[ECS] field.\n\n")

    if "index" in field:
        if not field["index"]:
            output.write("{}\n\n".format("Field is not indexed."))

    if "enabled" in field:
        if not field["enabled"]:
            output.write("{}\n\n".format("Object is not enabled."))

    output.write("\n")

    if "multi_fields" in field:
        for subfield in field["multi_fields"]:
            document_field(output, subfield, field_path + "." +
                           subfield["name"])

# Build the applies_to string: a comma-separated list
# of all available lifecycles and versions
# NOTE: There's almost certainly a more efficient way
# to accomplish this.


def get_applies_to(item):
    applies_to = []
    if "version" in item:
        if "preview" in item["version"]:
            applies_to.append("preview {}".format(item["version"]["preview"]))
        if "beta" in item["version"]:
            applies_to.append("beta {}".format(item["version"]["beta"]))
        if "ga" in item["version"]:
            applies_to.append("ga {}".format(item["version"]["ga"]))
        if "deprecated" in item["version"]:
            applies_to.append("deprecated {}".format(item["version"]["deprecated"]))
        if "removed" in item["version"]:
            applies_to.append("removed {}".format(item["version"]["removed"]))
    # Add `deprecated` to applies_to if not already added via `version`
    if "deprecated" in item:
        if "version" not in item or ("version" in item and "deprecated" not in item["version"]):
            applies_to.append("deprecated {}".format(item["deprecated"]))
    # Add `release` to applies_to if not already added via `version`
    if "release" in item:
        if "version" not in item and item["release"] != "ga":
            applies_to.append(item["release"])
    return applies_to


@lru_cache(maxsize=None)
def ecs_fields():
    """
    Fetch flattened ECS fields based on ECS 1.6.0 spec
    The result of this function is cached
    """
    url = "https://raw.githubusercontent.com/elastic/ecs/v1.6.0/generated/ecs/ecs_flat.yml"
    resp = requests.get(url)
    if resp.status_code != 200:
        raise ValueError(resp.content)
    return yaml.load(resp.content, Loader=yaml.FullLoader)


def fields_to_asciidoc(input, output_path, beat):
    output = open(os.path.join(output_path, "exported-fields.md"), 'w', encoding='utf-8')

    dict = {'beat': beat, 'title': beat.title()}

    output.write("""---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/{beat}/current/exported-fields.html
---

% This file is generated! See scripts/generate_fields_docs.py

# Exported fields [exported-fields]

This document describes the fields that are exported by {title}. They are grouped in the following categories:

""".format(**dict))

    docs = yaml.load(input, Loader=yaml.FullLoader)

    # fields file is empty
    if docs is None:
        print("fields.yml file is empty. fields.asciidoc cannot be generated.")
        return

    # deduplicate fields, last one wins
    for section in docs:
        if not section.get("fields"):
            continue
        fields = OrderedDict()
        for field in section["fields"]:
            name = field["name"]
            if name in fields:
                assert field["type"] == fields[name]["type"], 'field "{}" redefined with different type "{}"'.format(
                    name, field["type"])
                fields[name].update(field)
            else:
                fields[name] = field
        section["fields"] = list(fields.values())

    # Create sections from available fields
    sections = {}
    for v in docs:
        sections[v["key"]] = v["title"]

    for section in sorted(docs, key=lambda field: field["key"]):
        if "anchor" not in section:
            section["anchor"] = section["key"]

        if "skipdocs" not in section:
            output.write(
                "* [*{} fields*](/reference/{}/exported-fields-{}.md)\n".format(section["title"], beat, section["anchor"]))
    output.close()
    # Sort alphabetically by key
    for section in sorted(docs, key=lambda field: field["key"]):
        section["name"] = section["title"]
        if "anchor" not in section:
            section["anchor"] = section["key"]
        if "description" not in section:
            section["description"] = section["key"]
        if "fields" not in section or not section["fields"]:
            continue
        output_fields = open(os.path.join(
            output_path, "exported-fields-{}.md".format(section["anchor"])), 'w', encoding='utf-8')
        document_fields(output_fields, section, sections, "", beat)


if __name__ == "__main__":

    parser = argparse.ArgumentParser(
        description="Generates the documentation for a Beat.")
    parser.add_argument("fields", help="Path to fields.yml")
    parser.add_argument("beattitle", help="The beat title")
    parser.add_argument("es_beats",
                        help="The path to the general beats folder")
    parser.add_argument("--output_path", default="", dest="output_path",
                        help="Output path, if different from path")

    args = parser.parse_args()

    fields_yml = args.fields
    beat_title = args.beattitle
    es_beats = args.es_beats

    # Read fields.yml
    with open(fields_yml, encoding='utf-8') as f:
        fields = f.read()

    fields_to_asciidoc(fields, args.output_path, beat_title)
