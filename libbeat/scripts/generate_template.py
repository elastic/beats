#!/usr/bin/env python

"""
This script generates the ES template file (packetbeat.template.json) from
the _meta/fields.yml file.

Example usage:

   python generate_template.py filebeat/ filebeat
"""

import yaml
import json
import argparse
import os


def fields_to_es_template(args, input, output, index, version):
    """
    Reads the YAML file from input and generates the JSON for
    the ES template in output. input and output are both file
    pointers.
    """

    # Custom properties
    docs = yaml.load(input)

    # No fields defined, can't generate template
    if docs is None:
        print("fields.yml is empty. Cannot generate template.")
        return

    # Each template needs defaults
    if "defaults" not in docs.keys():
        print("No defaults are defined. Each template needs at" +
              " least defaults defined.")
        return

    defaults = docs["defaults"]

    for k, section in enumerate(docs["fields"]):
        docs["fields"][k] = dedot(section)

    # skeleton
    template = {
        "template": index,
        "order": 0,
        "settings": {
            "index.refresh_interval": "5s"
        },
        "mappings": {
            "_default_": {
                "_all": {
                    "norms": False
                },
                "properties": {},
                "_meta": {
                    "version": version,
                }
            }
        }
    }

    if args.es2x:
        # different syntax for norms
        template["mappings"]["_default_"]["_all"]["norms"] = {
            "enabled": False
        }

    properties = {}
    dynamic_templates = []

    # Make strings keywords by default
    if args.es2x:
        dynamic_templates.append({
            "strings_as_keyword": {
                "mapping": {
                    "type": "string",
                    "index": "not_analyzed",
                    "ignore_above": 1024
                },
                "match_mapping_type": "string",
            }
        })
    else:
        dynamic_templates.append({
            "strings_as_keyword": {
                "mapping": {
                    "type": "keyword",
                    "ignore_above": 1024
                },
                "match_mapping_type": "string",
            }
        })

    for section in docs["fields"]:
        prop, dynamic = fill_section_properties(args, section,
                                                defaults, "")
        properties.update(prop)
        dynamic_templates.extend(dynamic)

    template["mappings"]["_default_"]["properties"] = properties
    if len(dynamic_templates) > 0:
        template["mappings"]["_default_"]["dynamic_templates"] = \
            dynamic_templates

    json.dump(template, output,
              indent=2, separators=(',', ': '),
              sort_keys=True)


def dedot(group):
    """
    Walk the tree and transform keys like "beat.name" and "beat.hostname" into
      - name: "beat"
        type: group
        fields:
        - name: name
            ...
        - name: hostname
            ...
    """
    fields = []
    dedotted = {}

    for field in group["fields"]:
        if "." in field["name"]:
            # dedot
            newkey, rest = field["name"].split(".", 1)
            field["name"] = rest    # move one level down
            if newkey not in dedotted:
                dedotted[newkey] = {
                    "name": newkey,
                    "type": "group",
                    "fields": []
                }
            if field.get("type") == "group":
                dedotted[newkey]["fields"].append(dedotted(field))
            else:
                dedotted[newkey]["fields"].append(field)
        elif field.get("type") == "group":
            # call recursively
            fields.append(dedot(field))
        else:
            fields.append(field)
    for _, field in dedotted.items():
        fields.append(dedot(field))
    group["fields"] = fields
    return group


def fill_section_properties(args, section, defaults, path):
    """
    Traverse the sections tree and fill in the properties
    map.
    """
    properties = {}
    dynamic_templates = []

    for field in section["fields"]:
        prop, dynamic = fill_field_properties(args, field, defaults, path)
        properties.update(prop)
        dynamic_templates.extend(dynamic)

    return properties, dynamic_templates


def fill_field_properties(args, field, defaults, path):
    """
    Add data about a particular field in the properties
    map.
    """
    properties = {}
    dynamic_templates = []

    for key in defaults.keys():
        if key not in field:
            field[key] = defaults[key]

    if field["type"] == "text":
        if args.es2x:
            properties[field["name"]] = {
                "type": "string",
                "index": "analyzed",
                "norms": {
                    "enabled": False
                }
            }
        else:
            properties[field["name"]] = {
                "type": field["type"],
                "norms": False
            }

    elif field["type"] == "keyword":
        if args.es2x:
            properties[field["name"]] = {
                "type": "string",
                "index": "not_analyzed",
                "ignore_above": 1024
            }
        else:
            properties[field["name"]] = {
                "type": "keyword",
                "ignore_above": 1024
            }

    elif field["type"] in ["geo_point", "date", "long", "integer",
                           "double", "float", "half_float", "scaled_float",
                           "boolean"]:
        # Convert all integer fields to long
        if field["type"] == "integer":
            field["type"] = "long"

        if args.es2x and field["type"] in ["half_float", "scaled_float"]:
            # ES 2.x doesn't support half or scaled floats, so convert to float
            field["type"] = "float"

        properties[field["name"]] = {
            "type": field.get("type")
        }

        if field["type"] == "scaled_float":
            properties[field["name"]]["scaling_factor"] = \
                field.get("scaling_factor", 1000)

    elif field["type"] in ["dict", "list"]:
        if field.get("dict-type") == "text":
            # add a dynamic template to set all members of
            # the dict as text
            if len(path) > 0:
                name = path + "." + field["name"]
            else:
                name = field["name"]

            if args.es2x:
                dynamic_templates.append({
                    name: {
                        "mapping": {
                            "type": "string",
                            "index": "analyzed",
                        },
                        "match_mapping_type": "string",
                        "path_match": name + ".*"
                    }
                })
            else:
                dynamic_templates.append({
                    name: {
                        "mapping": {
                            "type": "text",
                        },
                        "match_mapping_type": "string",
                        "path_match": name + ".*"
                    }
                })

    elif field.get("type") == "group":
        if len(path) > 0:
            path = path + "." + field["name"]
        else:
            path = field["name"]
        prop, dynamic = fill_section_properties(args, field, defaults, path)

        # Only add properties if they have a content
        if len(prop) is not 0:
            properties[field.get("name")] = {"properties": {}}
            properties[field.get("name")]["properties"] = prop

        dynamic_templates.extend(dynamic)
    elif field.get("type") == "nested":
        if len(path) > 0:
            path = path + "." + field["name"]
        else:
            path = field["name"]
        prop, dynamic = fill_section_properties(args, field, defaults, path)

        # Only add properties if they have a content
        if len(prop) is not 0:
            properties[field.get("name")] = {
                "type": "nested",
                "properties": {}
            }
            properties[field.get("name")]["properties"] = prop

        dynamic_templates.extend(dynamic)
    else:
        raise ValueError("Unkown type found: " + field.get("type"))

    return properties, dynamic_templates


if __name__ == "__main__":

    parser = argparse.ArgumentParser(
        description="Generates the templates for a Beat.")
    parser.add_argument("--es2x", action="store_true",
                        help="Generate template for Elasticsearch 2.x.")
    parser.add_argument("path", help="Path to the beat folder")
    parser.add_argument("beatname", help="The beat fname")
    parser.add_argument("es_beats", help="The path to the general beats folder")

    args = parser.parse_args()

    target = args.path + "/" + args.beatname + ".template"
    if args.es2x:
        target += "-es2x"
    target += ".json"

    fields_yml = args.path + "/_meta/fields.generated.yml"

    # Not all beats have a fields.generated.yml. Fall back to fields.yml
    if not os.path.isfile(fields_yml):
        fields_yml = args.path + "/_meta/fields.yml"

    with open(fields_yml, 'r') as f:
        fields = f.read()

        # Prepend beat fields from libbeat
        with open(args.es_beats + "/libbeat/_meta/fields.generated.yml") as f:
            fields = f.read() + fields

        with open(args.es_beats + "/dev-tools/packer/version.yml") as file:
            version_data = yaml.load(file)

        with open(target, 'w') as output:
            fields_to_es_template(args, fields, output, args.beatname + "-*", version_data['version'])
