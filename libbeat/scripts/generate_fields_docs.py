import yaml
import os
import argparse


def document_fields(output, section, sections, path):
    if "anchor" in section:
        output.write("[[exported-fields-{}]]\n".format(section["anchor"]))

    if "prefix" in section:
        output.write("{}\n".format(section["prefix"]))

    # Intermediate level titles
    if "description" in section and "prefix" not in section and "anchor" not in section:
        output.write("[float]\n")

    if "description" in section:
        output.write("== {} fields\n\n".format(section["name"]))
        output.write("{}\n\n".format(section["description"]))

    if "fields" not in section or not section["fields"]:
        return

    output.write("\n")
    for field in section["fields"]:

        # Skip entries which do not define a name
        if "name" not in field:
            continue

        if path == "":
            newpath = field["name"]
        else:
            newpath = path + "." + field["name"]

        if "type" in field and field["type"] == "group":
            document_fields(output, field, sections, newpath)
        else:
            document_field(output, field, newpath)


def document_field(output, field, path):

    if "path" not in field:
        field["path"] = path

    output.write("*`{}`*::\n+\n--\n".format(field["path"]))

    if "type" in field:
        output.write("type: {}\n\n".format(field["type"]))
    if "example" in field:
        output.write("example: {}\n\n".format(field["example"]))
    if "format" in field:
        output.write("format: {}\n\n".format(field["format"]))
    if "required" in field:
        output.write("required: {}\n\n".format(field["required"]))

    if "description" in field:
        output.write("{}\n\n".format(field["description"]))

    if "index" in field:
        if not field["index"]:
            output.write("{}\n\n".format("Field is not indexed."))

    if "enable" in field:
        if not field["enable"]:
            output.write("{}\n\n".format("Object is not enabled."))

    if "multi_fields" in field:
        for subfield in field["multi_fields"]:
            document_field(output, subfield, path + "." + subfield["name"])
    output.write("--\n\n")


def fields_to_asciidoc(input, output, beat):

    dict = {'beat': beat}

    output.write("""
////
This file is generated! See _meta/fields.yml and scripts/generate_field_docs.py
////

[[exported-fields]]
= Exported fields

[partintro]

--
This document describes the fields that are exported by {beat}. They are
grouped in the following categories:

""".format(**dict))

    docs = yaml.load(input)

    # fields file is empty
    if docs is None:
        print("fields.yml file is empty. fields.asciidoc cannot be generated.")
        return

    # Create sections from available fields
    sections = {}
    for v in docs:
        sections[v["key"]] = v["title"]

    for section in sorted(docs, key=lambda field: field["key"]):
        if "anchor" not in section:
            section["anchor"] = section["key"]

        output.write("* <<exported-fields-{}>>\n".format(section["anchor"]))
    output.write("\n--\n")

    # Sort alphabetically by key
    for section in sorted(docs, key=lambda field: field["key"]):
        section["name"] = section["title"]
        if "anchor" not in section:
            section["anchor"] = section["key"]
        document_fields(output, section, sections, "")


if __name__ == "__main__":

    parser = argparse.ArgumentParser(
        description="Generates the documentation for a Beat.")
    parser.add_argument("path", help="Path to the beat folder")
    parser.add_argument("beattitle", help="The beat title")
    parser.add_argument("es_beats", help="The path to the general beats folder")

    args = parser.parse_args()

    beat_path = args.path
    beat_title = args.beattitle
    es_beats = args.es_beats

    fields_yml = beat_path + "/fields.yml"

    # Read fields.yml
    with open(fields_yml) as f:
        fields = f.read()

    output = open(beat_path + "/docs/fields.asciidoc", 'w')

    try:
        fields_to_asciidoc(fields, output, beat_title)
    finally:
        output.close()
