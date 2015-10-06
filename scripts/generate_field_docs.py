#!/usr/bin/env python

"""
This script generates markdown documentation from the fields yml file.

Usage: python generate_field_docs.py file.yml file.asciidoc
"""

import sys
import yaml

SECTIONS = [
    ("event", "Event"),
    ("dns", "DNS"),
    ("http", "Http"),
    ("memcache", "Memcache"),
    ("mysql", "Mysql"),
    ("pgsql", "PostgreSQL"),
    ("thrift", "Thrift-RPC"),
    ("redis", "Redis"),
    ("mongodb", "MongoDb"),
    ("measurements", "Measurements"),
    ("env", "Environmental"),
    ("raw", "Raw")]


def document_fields(output, section):

    if "anchor" in section:
        output.write("[[exported-fields-{}]]\n".format(section["anchor"]))
    output.write("=== {} fields\n\n".format(section["name"]))

    if "description" in section:
        output.write("{}\n\n".format(section["description"]))

    output.write("\n")
    for field in section["fields"]:

        if "type" in field and field["type"] == "group":
            for sec, name in SECTIONS:
                if sec == field["name"]:
                    field["anchor"] = field["name"]
                    field["name"] = name
                    break
            document_fields(output, field)
        else:
            document_field(output, field)


def document_field(output, field):

    output.write("==== {}\n\n".format(field["name"]))

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


def fields_to_asciidoc(input, output):

    output.write("""
////
This file is generated! See etc/fields.yml and scripts/generate_field_docs.py
////

[[exported-fields]]
== Exported fields

This document describes the fields that are exported by the
Packetbeat shipper for each transaction. They are grouped in the
following categories:

""")

    for doc, _ in SECTIONS:
        output.write("* <<exported-fields-{}>>\n".format(doc))
    output.write("\n")

    docs = yaml.load(input)

    for doc, name in SECTIONS:
        if doc in docs:
            section = docs[doc]
            if "type" in section:
                if section["type"] == "group":
                    section["name"] = name
                    section["anchor"] = doc
                    document_fields(output, section)


if __name__ == "__main__":
    if len(sys.argv) != 3:
        print "Usage: %s file.yml file.md" % sys.argv[0]
        sys.exit(1)

    input = open(sys.argv[1], 'r')
    output = open(sys.argv[2], 'w')

    try:
        fields_to_asciidoc(input, output)
    finally:
        input.close()
        output.close()
