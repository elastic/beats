import yaml
import sys

def document_fields(output, section, sections, path):
    if "anchor" in section:
        output.write("[[exported-fields-{}]]\n".format(section["anchor"]))

    if "prefix" in section:
        output.write("{}\n".format(section["prefix"]))

    if "description" in section:
        output.write("=== {} Fields\n\n".format(section["name"]))
        output.write("{}\n\n".format(section["description"]))

    if "fields" not in section or not section["fields"]:
        return

    output.write("\n")
    for field in section["fields"]:

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

    output.write("==== {}\n\n".format(field["path"]))

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


def fields_to_asciidoc(input, output, beat):

    dict = {'beat': beat}

    output.write("""
////
This file is generated! See etc/fields.yml and scripts/generate_field_docs.py
////

[[exported-fields]]
== Exported Fields

This document describes the fields that are exported by {beat}. They are
grouped in the following categories:

""".format(**dict))


    docs = yaml.load(input)

    # fields file is empty
    if docs is None:
        print "fields.yml file is empty. fields.asciidoc cannot be generated."
        return

    # Create sections from available fields
    sections = {}
    for v in docs["fields"]:
        sections[v["key"]] = v["title"]

    for key in sorted(sections):
        output.write("* <<exported-fields-{}>>\n".format(key))
    output.write("\n")

    # Sort alphabetically by key
    for section in sorted(docs["fields"], key=lambda field: field["key"]):
        section["name"] = section["title"]
        section["anchor"] = section["key"]
        document_fields(output, section, sections, "")


if __name__ == "__main__":

    if len(sys.argv) != 3:
        print "Usage: %s beatpath beatname" % sys.argv[0]
        sys.exit(1)

    beat_path = sys.argv[1]
    beat_name = sys.argv[2]

    # Read fields.yml
    with file(beat_path + "/etc/fields.yml") as f:
        fields = f.read()

    # Prepends beat fields from libbeat
    with file(beat_path + "/../libbeat/_beat/fields.yml") as f:
        fields = f.read() + fields

    output = open(beat_path + "/docs/fields.asciidoc", 'w')

    try:
        fields_to_asciidoc(fields, output, beat_name.title())
    finally:
        output.close()
