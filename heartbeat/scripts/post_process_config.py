#!/usr/bin/env python

# This script is invoked from libbeat/scripts/Makefile after heartbeat.yml has been generated
# It rewrites that file to remove the standard 'Processors' section

import sys
import re

result = ""

# Each section is formatted something like #==== Section Name ====
# which is captured by this regex with the name as the first group
start_section = re.compile(r'^#[=]+\s+?([\w\s]+)\s+?[=]+$')

output = ""
inside_processor_section = False

# Filter out the Processor sectiuon
with open(sys.argv[1], 'r') as conf_file:
    for line in conf_file:
        m = start_section.match(line)
        if m:
            section_name = m.group(1)
            if section_name == "Processors":
                inside_processor_section = True
            else:
                inside_processor_section = False
        if not inside_processor_section:
            output += line


with open(sys.argv[1], 'w') as conf_file:
    conf_file.write(output)
