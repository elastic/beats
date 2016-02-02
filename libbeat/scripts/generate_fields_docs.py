import yaml
import sys


def document_fields(output, section, sections):

    if "anchor" in section:
        output.write("[[exported-fields-{}]]\n".format(section["anchor"]))
    output.write("=== {} Fields\n\n".format(section["name"]))

    if "description" in section:
        output.write("{}\n\n".format(section["description"]))

    if "fields" not in section or not section["fields"]:
        return

    output.write("\n")
    for field in section["fields"]:

        if "type" in field and field["type"] == "group":

            for value in sections:
                sec = value[0]
                name = value[1]
                if sec == field["name"]:
                    field["anchor"] = field["name"]
                    field["name"] = name
                    break
            document_fields(output, field, sections)
        else:
            document_field(output, field)


def document_field(output, field):

    if "path" not in field:
        field["path"] = field["name"]

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
    sections = docs["sections"]

    for doc, _ in sections:
        output.write("* <<exported-fields-{}>>\n".format(doc))
    output.write("\n")

    for value in sections:

        doc = value[0]
        name = value[1]

        if doc in docs:
            section = docs[doc]
            if "type" in section:
                if section["type"] == "group":
                    section["name"] = name
                    section["anchor"] = doc
                    document_fields(output, section, sections)


if __name__ == "__main__":

    if len(sys.argv) != 3:
        print "Usage: %s beatpath beatname" % sys.argv[0]
        sys.exit(1)

    beat_path = sys.argv[1]
    beat_name = sys.argv[2]

    input = open(beat_path + "/etc/fields.yml", 'r')
    output = open(beat_path + "/docs/fields.asciidoc", 'w')

    try:
        fields_to_asciidoc(input, output, beat_name.title())
    finally:
        input.close()
        output.close()
