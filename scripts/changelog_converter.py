#!/usr/bin/env bash

import re

# Converts the old CHANGELOG.asciidoc to the new changelog.yml file

beats = ["filebeat", "winlogbeat", "metricbeat", "heartbeat", "packetbeat", "libbeat"]

def parse_config(lines):

    file = ""

    for beat in beats:
        file = open("../" + beat + "/changelog.yml", "w")
        file.write("""project: """ + beat + """
changes:""")

    beat = ""
    type = ""

    for line in lines:

        if beat is not "":
            file = open("../" + beat + "/changelog.yml", "a")

        # Match beat
        result = re.search("^\*(.*)\*", line)
        if result:
            beat = result.group(1)

            if beat == "Affecting all Beats":
                beat = "libbeat"
            elif beat == "Topbeat":
                beat = "topbeat"
            elif beat == "Filebeat":
                beat = "filebeat"
            elif beat == "Metricbeat":
                beat = "metricbeat"
            elif beat == "Packetbeat":
                beat = "packetbeat"
            elif beat == "Packetbeat":
                beat = "packetbeat"
            elif beat == "Winlogbeat":
                beat = "winlogbeat"
            elif beat == "libbeat":
                beat = "libbeat"
            else:
                print "BEAT: " + beat

            continue


        # Match type
        result = re.search("==== (.*)", line)
        if result:
            type = result.group(1)

            if type == "Bugfixes":
                type = "bug"
            elif type == "Added":
                type = "added"
            elif type == "Breaking changes":
                type = "breaking"
            elif type == "Deprecated":
                type = "deprecated"
            elif type == "Known issues":
                type = "known_issues"
            else:
                print "TYPE: " + type

            continue


        # Match releases
        result = re.search("https://github.com/elastic/beats/compare/(.*)\.\.\.(.*)\[(.*)", line)
        if result:
            previous = result.group(1)
            current = result.group(2)
            release = """
- type: release
  version: """ + current + """
  parent: """ + previous + """
  date: ""
"""
            for beat in beats:
                file = open("../" + beat + "/changelog.yml", "a")
                file.write(release)
            continue

        # Match issue
        result = re.search("- (.*) \{[a-z]*\}([0-9]*)", line)
        if result:
            text = result.group(1)
            issue = result.group(2)
            issueBlock = """
- type: """ + type + """
  issue: """ + issue + """
  description: >
    """ + text + """
"""

            file.write(issueBlock +"\n")
            continue

        #- Fix kafka output re-trying batches with too large events. {issue}2735[2735]

        # Create initial file
        # Search lines with     https://github.com/elastic/beats/compare/v5.0.0-beta1...v5.0.0-rc1
        # Write release inside



if __name__ == "__main__":

    # Ignore name of script

    # Skip listing project if only one changelog

    with open("../CHANGELOG.asciidoc") as f:
        content = f.readlines()

    parse_config(content)


#project: beats
#changes:
#
#- type: added
#  description: >
#  Add changelog generation script to generate changelogs from structured yaml files.
#  issue: 1879
#  target: dev
#
#- type: release
#  version: 5.0.0-alpha3
#  parent: 5.0.0-alpha2
#  date: "2016-05-31"


