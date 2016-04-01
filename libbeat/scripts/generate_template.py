#!/usr/bin/env python

"""
This script generates the ES template file (topbeat.template.json) from
the etc/fields.yml file.

Example usage:

   python generate_template.py etc/fields.yml topbeat.template.json
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
        print "No defaults are defined. Each template needs at least defaults defined."
        return

    defaults = docs["defaults"]

    # skeleton
    template = {
        "template": index,
        "settings": {
            "index.refresh_interval": "5s"
        },
        "mappings": {
            "_default_": {
                "_all": {
                    "norms": False
                },
                "properties": {},
                "dynamic_templates": [{
                    "template1": {
                        "match_mapping_type": "string",
                        "mapping": {
                            "type": "keyword",
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
            prop = fill_section_properties(section, defaults)
            properties.update(prop)

    template["mappings"]["_default_"]["properties"] = properties

    json.dump(template, output,
              indent=2, separators=(',', ': '),
              sort_keys=True)


def fill_section_properties(section, defaults):
    """
    Traverse the sections tree and fill in the properties
    map.
    """
    properties = {}

    for field in section["fields"]:
        prop = fill_field_properties(field, defaults)
        properties.update(prop)

    return properties


def fill_field_properties(field, defaults):
    """
    Add data about a particular field in the properties
    map.
    """
    properties = {}

    for key in defaults.keys():
        if key not in field:
            field[key] = defaults[key]

    # TODO: Make this more dyanmic
    if field.get("type") == "text":
        properties[field["name"]] = {
            "type": field["type"],
            "norms": False
        }

    elif field.get("type")  == "keyword":
        mapping = {
        }
        if field.get("doc_values") != defaults.get("doc_values"):
             mapping["doc_values"] = field.get("doc_values")
        if field.get("ignore_above") != defaults.get("ignore_above"):
             mapping["ignore_above"] = field.get("ignore_above")
        if len(mapping) > 0:
            # only add the field if the mapping is different from the defaults
            mapping["type"] = "keyword"
            properties[field["name"]] = mapping

    elif field.get("type") in ["geo_point", "date", "long", "integer", "double", "float"]:
        properties[field["name"]] = {
            "type": field.get("type")
        }

    elif field.get("type") == "group":
        prop = fill_section_properties(field, defaults)

        # Only add properties if they have a content
        if len(prop) is not 0:
            properties[field.get("name")] = {"properties": {}}
            properties[field.get("name")]["properties"] = prop

    # Otherwise let dynamic mappings do the job

    return properties


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
