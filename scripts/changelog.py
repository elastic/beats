#!/usr/bin/env python


# TODO
# * Document
# * Implement target in creation
# * Implement filtering based on type (user / dev). Define which is default type (user?)

# Notes
# * RELEASE must be introduced in all changelog files. Advantage is that on the top level if only change in one repo
#   new release will be generated automatically. Release name is used for syncing across repos.
# Everything is put into a global array which is `changelog[release][beat][type] = change`

import yaml
import sys
import argparse

current_version = "unreleased"
sorted_changelog = {}


def load_files(files):

    global current_version

    # Reset current version after each file
    current_version = "unreleased"

    for file in files:
        with open(file) as f:
            content = f.read()
        parse(content)


def parse(content):

    global sorted_changelog

    if content == "":
        return

    changelog = yaml.load(content)
    project = changelog['project']

    if changelog['changes'] is not None:
        for entry in changelog['changes']:
            add(entry, project)


# Adds an entry to the global changelog array
def add(entry, project):
    global current_version
    global sorted_changelog

    if entry['type'] == 'release':
        current_version = entry['version']

    if current_version not in sorted_changelog:
        sorted_changelog[current_version] = {}
        sorted_changelog[current_version]["details"] = []

    if project not in sorted_changelog[current_version]:
        sorted_changelog[current_version][project] = {}

    if entry["type"] not in sorted_changelog[current_version][project] and entry["type"] != "release":
        sorted_changelog[current_version][project][entry["type"]] = []

    if entry["type"] == "release":
        sorted_changelog[current_version]["details"].append(entry)
    else:
        # Set user as the default target
        if "target" not in entry:
            entry["target"] = "user"
        sorted_changelog[current_version][project][entry["type"]].append(entry)


# Generates the asciidoc changelog from the changelog entries
def output_asciidoc(skip_project, target):

    global sorted_changelog

    header = """////
This file is generated! See scripts/changelog.py
////"""
    print header


    # Make sure newest release is on top
    releases = sorted(sorted_changelog, reverse=True)
    c = ""

    for release in releases:
        # Print out version details, take first entry
        c += parse_release(sorted_changelog[release]["details"])
        del sorted_changelog[release]["details"]

        for key, project in sorted_changelog[release].iteritems():
            if not skip_project:
                c += parse_project(key)
            for key, type in project.iteritems():

                type_content = parse_type(key)
                entry_content = ""

                for entry in type:
                    entry_content += parse_entry(entry, target)

                # Only add title and content if there are entries
                if entry_content != "":
                    c += type_content + entry_content

    return c


# Creates ascii doc for the given entry
def parse_release(release):

    if len(release) == 0:
        return "\n=== Unreleased\n\n"

    # TODO: This could be made more sophisticated -> pick one with description or concatenate all together?
    entry = release[0]
    c = """
[[release-notes-""" + entry['version'] + """]]
=== Beats version """ + entry['version'] + """
""" + entry['date'] + """ https://github.com/elastic/beats/compare/""" + entry['parent'] + "..." + entry['version'] + """[View commits]"""
    return c + "\n"

# Parses an entry to asciidoc
def parse_entry(entry, target):

    if target != "" and target != entry["target"]:
        return ""

    c = "- " + entry['description']

    if "issue" in entry:
        issue = str(entry["issue"])
        c += "  {issue}" + issue + "[" + issue + "]"

    c += "\n"
    return c

def parse_type(type):
    return "\n==== " + type.title() + "\n\n"

def parse_project(project):
    return "\n*" + project.title() + "*\n\n"



if __name__ == "__main__":
    args = sys.argv

    parser = argparse.ArgumentParser(
        description="Generates the templates for a Beat.")
    parser.add_argument('changelogs', metavar='changelog.yml', nargs='+',
                        help='List of changelog files')
    parser.add_argument("--target", help="Choose user or dev as target. If target is defined, both are included", default="")

    args = parser.parse_args()
    vars = vars(args)

    # Ignore name of script
    load_files(vars["changelogs"])

    # Skip listing project if only one changelog
    skip_project = len(vars["changelogs"]) == 1

    print output_asciidoc(skip_project, vars["target"])
