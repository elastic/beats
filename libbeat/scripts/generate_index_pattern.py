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


def fields_to_json(section, path, output):

    for field in section["fields"]:
        if path == "":
            newpath = field["name"]
        else:
            newpath = path + "." + field["name"]

        if "type" in field and field["type"] == "group":
            fields_to_json(field, newpath, output)
        else:
            field_to_json(field, newpath, output)


def field_to_json(desc, path, output):

    field = {
        "name": path,
        "count": 0,
        "scripted": False,
        "indexed": True,
        "analyzed": False,
        "doc_values": True,
    }
    if "type" in desc:
        if desc["type"] in ["half_float", "float", "integer", "long"]:
            field["type"] = "number"
        else:
            field["type"] = desc["type"]
    else:
        field["type"] = "string"

    output["fields"].append(field)

    if "format" in desc:
        output["fieldFormatMap"][path] = {
            "id": desc["format"],
        }


def fields_to_index_pattern(args, input):

    docs = yaml.load(input)

    if docs is None:
        print("fields.yml is empty. Cannot generate index-pattern")
        return

    output = {
        "fields": [],
        "fieldFormatMap": {},
        "timeFieldName": "@timestamp",
        "title": args.index,

    }

    for k, section in enumerate(docs["fields"]):
        fields_to_json(section, "", output)

    output["fields"] = json.dumps(output["fields"])
    output["fieldFormatMap"] = json.dumps(output["fieldFormatMap"])
    return output


def get_index_pattern_name(index):

    allow = string.letters + string.digits + "_"
    return re.sub('[^%s]' % allow, '', index)


if __name__ == "__main__":

    parser = argparse.ArgumentParser(
        description="Generates the index-pattern for a Beat.")
    parser.add_argument("--index", help="The name of the index-pattern")
    parser.add_argument("--beat", help="Local Beat directory")
    parser.add_argument("--libbeat", help="Libbeat local directory")

    args = parser.parse_args()

    # generate the index-pattern content
    with open(args.beat + "/etc/fields.yml", 'r') as f:
        fields = f.read()

        # Prepend beat fields from libbeat
        with open(args.libbeat + "/_meta/fields.yml") as f:
            fields = f.read() + fields

        # with open(target, 'w') as output:
        output = fields_to_index_pattern(args, fields)

    # dump output to a json file
    fileName = get_index_pattern_name(args.index)
    target = args.beat + "/etc/kibana/index-pattern/" + fileName + ".json"

    output = json.dumps(output, indent=2)

    with open(target, 'w') as f:
        f.write(output)

    print "The index pattern was created under {}".format(target)
