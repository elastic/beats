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

    global unique_fields

    if path in unique_fields:
        print "ERROR: Field", path, "is duplicated. Please delete it and try again. Fields already are", unique_fields
        sys.exit(1)
    else:
        unique_fields.append(path)

    field = {
        "name": path,
        "count": 0,
        "scripted": False,
        "indexed": True,
        "analyzed": False,
        "doc_values": True,
        "searchable": True,
        "aggregatable": True,
    }
    # find the kibana types based on the field type
    if "type" in desc:
        if desc["type"] in ["half_float", "scaled_float", "float", "integer", "long"]:
            field["type"] = "number"
        elif desc["type"] in ["text", "keyword"]:
            field["type"] = "string"
            if desc["type"] == "text":
                field["aggregatable"] = False
        elif desc["type"] == "date":
            field["type"] = "date"
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

    allow = string.ascii_letters + string.digits + "_"
    return re.sub('[^%s]' % allow, '', index)


if __name__ == "__main__":

    parser = argparse.ArgumentParser(
        description="Generates the index-pattern for a Beat.")
    parser.add_argument("--index", help="The name of the index-pattern")
    parser.add_argument("--beat", help="Local Beat directory")
    parser.add_argument("--libbeat", help="Libbeat local directory")

    args = parser.parse_args()

    fields_yml = args.beat + "/_meta/fields.generated.yml"

    # Not all beats have a fields.generated.yml. Fall back to fields.yml
    if not os.path.isfile(fields_yml):
        fields_yml = args.beat + "/_meta/fields.yml"

    # generate the index-pattern content
    with open(fields_yml, 'r') as f:
        fields = f.read()

        # Prepend beat fields from libbeat
        with open(args.libbeat + "/_meta/fields.generated.yml") as f:
            fields = f.read() + fields

        # with open(target, 'w') as output:
        output = fields_to_index_pattern(args, fields)

    # dump output to a json file
    fileName = get_index_pattern_name(args.index)
    target_dir = os.path.join(args.beat, "_meta", "kibana", "index-pattern")
    target_file =os.path.join(target_dir, fileName + ".json")

    try: os.makedirs(target_dir)
    except OSError as exception:
        if exception.errno != errno.EEXIST: raise

    output = json.dumps(output, indent=2)

    with open(target_file, 'w') as f:
        f.write(output)

    print ("The index pattern was created under {}".format(target_file))
