#!/usr/bin/env python

"""
This script generates the ES template file (packetbeat.template.json) from
the etc/fields.yml file.

Example usage:

   python generate_template.py etc/fields.yml etc/packetbeat.template.json
"""

import sys
import yaml
import json


def fields_to_es_template(input, output):
    """
    Reads the YAML file from input and generates the JSON for
    the ES template in output. input and output are both file
    pointers.
    """

    # Custom properties
    docs = yaml.load(input)

    defaults = docs["defaults"]

    # skeleton
    template = {
        "template": "packetbeat-*",
        "settings": {
            "index.refresh_interval": "5s"
        },
        "mappings": {
            "_default_": {
                "_all": {
                    "enabled": True,
                    "norms": {
                        "enabled": False
                    }
                },
                "properties": {},
                "dynamic_templates": [{
                    "template1": {
                        "match": "*",
                        "mapping": {
                            "type": "{dynamic_type}",
                            "index": defaults["index"],
                            "doc_values": defaults["doc_values"],
                            "ignore_above": defaults["ignore_above"]
                        }
                    }
                }]
            }
        }
    }

    properties = {}
    for doc, section in docs.items():
        if doc not in ["version", "defaults"]:
            fill_section_properties(properties, section, defaults)

    template["mappings"]["_default_"]["properties"] = properties

    json.dump(template, output,
              indent=2, separators=(',', ': '),
              sort_keys=True)


def fill_section_properties(properties, section, defaults):
    """
    Traverse the sections tree and fill in the properties
    map.
    """
    for field in section["fields"]:
        if field.get("type") == "group":
            fill_section_properties(properties, field, defaults)
        else:
            fill_field_properties(properties, field, defaults)


def fill_field_properties(properties, field, defaults):
    """
    Add data about a particular field in the properties
    map.
    """
    for key in defaults.keys():
        if key not in field:
            field[key] = defaults[key]

    if field.get("index") == "analyzed":
        properties[field["name"]] = {
            "type": field["type"],
            "index": "analyzed",
            "norms": {
                "enabled": False
            }
        }

    elif field.get("type") == "geo_point":
        properties[field["name"]] = {
            "type": "geo_point"
        }

    elif field.get("type") == "date":
        properties[field["name"]] = {
            "type": "date"
        }

    elif field.get("ignore_above") == 0:
        properties[field["name"]] = {
            "type": field["type"],
            "index": field["index"],
            "doc_values": field["doc_values"]
        }

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print "Usage: %s fields.yml template.json" % sys.argv[0]
        sys.exit(1)

    input = open(sys.argv[1], 'r')
    output = open(sys.argv[2], 'w')

    try:
        fields_to_es_template(input, output)
    finally:
        input.close()
        output.close()
