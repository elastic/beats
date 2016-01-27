#!/usr/bin/env python

"""
This script generates markdown documentation from the fields yml file.

Usage: python generate_field_docs.py file.yml file.asciidoc
"""

import os
import sys

sys.path.append(os.path.dirname(__file__) + '/../../libbeat/scripts')

import generate_fields_docs



SECTIONS = [
    ("event", "Event"),
    ("icmp", "ICMP"),
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


if __name__ == "__main__":
    if len(sys.argv) != 3:
        print "Usage: %s file.yml file.asciidoc" % sys.argv[0]
        sys.exit(1)

    input = open(sys.argv[1], 'r')
    output = open(sys.argv[2], 'w')

    try:
        generate_fields_docs.fields_to_asciidoc(input, output, SECTIONS, "Packetbeat")
    finally:
        input.close()
        output.close()
