#!/usr/bin/env python

"""
This script generates the index-pattern for Kibana from
the fields.yml file.
"""

import yaml
import argparse
import string
import re
import json
import os
import errno
import sys

unique_fields = []


def fields_to_json(fields, section, path, output):

    # Need in case there are no fields
    if section["fields"] is None:
        section["fields"] = {}

    for field in section["fields"]:
        if path == "":
            newpath = field["name"]
        else:
            newpath = path + "." + field["name"]

        if "type" in field and field["type"] == "group":
            fields_to_json(fields, field, newpath, output)
        else:
            field_to_json(fields, field, newpath, output)


def field_to_json(fields, desc, path, output,
                  indexed=True, analyzed=False, doc_values=True,
                  searchable=True, aggregatable=True):

    if path in unique_fields:
        print("ERROR: Field {} is duplicated. Please delete it and try again. Fields already are {}".format(
            path, ", ".join(fields)))
        sys.exit(1)
    else:
        fields.append(path)

    field = {
        "name": path,
        "count": 0,
        "scripted": False,
        "indexed": indexed,
        "analyzed": analyzed,
        "doc_values": doc_values,
        "searchable": searchable,
        "aggregatable": aggregatable,
    }
    # find the kibana types based on the field type
    if "type" in desc:
        if desc["type"] in ["half_float", "scaled_float", "float", "integer", "long", "short", "byte"]:
            field["type"] = "number"
        elif desc["type"] in ["text", "keyword"]:
            field["type"] = "string"
            if desc["type"] == "text":
                field["aggregatable"] = False
        elif desc["type"] == "date":
            field["type"] = "date"
        elif desc["type"] == "geo_point":
            field["type"] = "geo_point"
    else:
        field["type"] = "string"

    output["fields"].append(field)

    if "format" in desc:
        output["fieldFormatMap"][path] = {
            "id": desc["format"],
        }


def fields_to_index_pattern(version, args, input):

    docs = yaml.load(input)
    fields = []

    if docs is None:
        print("fields.yml is empty. Cannot generate index-pattern")
        return

    attributes = {
        "fields": [],
        "fieldFormatMap": {},
        "timeFieldName": "@timestamp",
        "title": args.index,

    }

    for k, section in enumerate(docs):
        fields_to_json(fields, section, "", attributes)

    # add meta fields

    field_to_json(fields, {"name": "_id", "type": "keyword"}, "_id",
                  attributes,
                  indexed=False, analyzed=False, doc_values=False,
                  searchable=False, aggregatable=False)

    field_to_json(fields, {"name": "_type", "type": "keyword"}, "_type",
                  attributes,
                  indexed=False, analyzed=False, doc_values=False,
                  searchable=True, aggregatable=True)

    field_to_json(fields, {"name": "_index", "type": "keyword"}, "_index",
                  attributes,
                  indexed=False, analyzed=False, doc_values=False,
                  searchable=False, aggregatable=False)

    field_to_json(fields, {"name": "_score", "type": "integer"}, "_score",
                  attributes,
                  indexed=False, analyzed=False, doc_values=False,
                  searchable=False, aggregatable=False)

    attributes["fields"] = json.dumps(attributes["fields"])
    attributes["fieldFormatMap"] = json.dumps(attributes["fieldFormatMap"])

    if version == "5.x":
        return attributes

    return {
        "version": args.version,
        "objects": [{
            "type": "index-pattern",
            "id": args.index,
            "version": 1,
            "attributes": attributes,
        }]


    }


def get_index_pattern_name(index):

    allow = string.ascii_letters + string.digits + "_"
    return re.sub('[^%s]' % allow, '', index)


def dump_index_pattern(args, version, output):

    fileName = get_index_pattern_name(args.index)
    target_dir = os.path.join(args.beat, "_meta", "kibana", version, "index-pattern")
    target_file = os.path.join(target_dir, fileName + ".json")

    try:
        os.makedirs(target_dir)
    except OSError as exception:
        if exception.errno != errno.EEXIST:
            raise

    output = json.dumps(output, indent=2)

    with open(target_file, 'w') as f:
        f.write(output)

    print("The index pattern was created under {}".format(target_file))
    return target_file


if __name__ == "__main__":

    parser = argparse.ArgumentParser(
        description="Generates the index-pattern for a Beat.")
    parser.add_argument("--version", help="Beat version")
    parser.add_argument("--index", help="The name of the index-pattern")
    parser.add_argument("--beat", help="Local Beat directory")

    args = parser.parse_args()

    fields_yml = args.beat + "/fields.yml"

    # generate the index-pattern content
    with open(fields_yml, 'r') as f:
        fields = f.read()

        # with open(target, 'w') as output:
        output = fields_to_index_pattern("default", args, fields)
        dump_index_pattern(args, "default", output)

        output5x = fields_to_index_pattern("5.x", args, fields)
        dump_index_pattern(args, "5.x", output5x)
