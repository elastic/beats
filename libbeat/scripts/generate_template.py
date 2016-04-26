#!/usr/bin/env python

"""
This script generates the ES template file (topbeat.template.json) from
the etc/fields.yml file.

Example usage:

   python generate_template.py filebeat/ filebeat
"""

import sys
import yaml
import json


def fields_to_es_template(input, output, index):
    """
    Reads the YAML file from input and generates the JSON for
    the ES template in output. input and output are both file
    pointers.
    """

    # Custom properties
    docs = yaml.load(input)

    # No fields defined, can't generate template
    if docs is None:
        print "fields.yml is empty. Cannot generate template."
        return

    # Remove sections as only needed for docs
    if "sections" in docs.keys():
        del docs["sections"]

    # Each template needs defaults
    if "defaults" not in docs.keys():
        print("No defaults are defined. Each template needs at" +
              " least defaults defined.")
        return

    defaults = docs["defaults"]

    # de-dot
    for doc, section in docs.items():
        if doc != "defaults":
            docs[doc] = dedot(section)

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
                "properties": {}
            }
        }
    }

    properties = {}
    dynamic_templates = []
    for doc, section in docs.items():
        if doc != "defaults":
            prop, dynamic = fill_section_properties(section, defaults, "")
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
        fields.append(field)
    group["fields"] = fields
    return group


def fill_section_properties(section, defaults, path):
    """
    Traverse the sections tree and fill in the properties
    map.
    """
    properties = {}
    dynamic_templates = []

    for field in section["fields"]:
        prop, dynamic = fill_field_properties(field, defaults, path)
        properties.update(prop)
        dynamic_templates.extend(dynamic)

    return properties, dynamic_templates


def fill_field_properties(field, defaults, path):
    """
    Add data about a particular field in the properties
    map.
    """
    properties = {}
    dynamic_templates = []

    for key in defaults.keys():
        if key not in field:
            field[key] = defaults[key]

    # TODO: Make this more dyanmic
    if field["type"] == "text":
        properties[field["name"]] = {
            "type": field["type"],
            "norms": False
        }

    elif field["type"] in ["geo_point", "date", "long", "integer",
                           "double", "float", "keyword"]:
        properties[field["name"]] = {
            "type": field.get("type")
        }
        if field["type"] == "keyword":
            properties[field["name"]]["ignore_above"] = \
                defaults.get("ignore_above", 1024)

    elif field["type"] == "dict":
        if field.get("dict-type") == "keyword":
            # add a dynamic template to set all members of
            # the dict as keywords
            if len(path) > 0:
                name = path + "." + field["name"]
            else:
                name = field["name"]

            dynamic_templates.append({
                name: {
                    "mapping": {
                        "index": True,
                        "type": "keyword",
                        "ignore_above": 1024
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
        prop, dynamic = fill_section_properties(field, defaults, path)

        # Only add properties if they have a content
        if len(prop) is not 0:
            properties[field.get("name")] = {"properties": {}}
            properties[field.get("name")]["properties"] = prop

        dynamic_templates.extend(dynamic)

    return properties, dynamic_templates


if __name__ == "__main__":

    if len(sys.argv) != 3:
        print "Usage: %s beatpath beatname" % sys.argv[0]
        sys.exit(1)

    beat_path = sys.argv[1]
    beat_name = sys.argv[2]

    input = open(beat_path + "/etc/fields.yml", 'r')
    output = open(beat_path + "/" + beat_name + ".template.json", 'w')

    try:
        fields_to_es_template(input, output, beat_name + "-*")
    finally:
        input.close()
        output.close()
