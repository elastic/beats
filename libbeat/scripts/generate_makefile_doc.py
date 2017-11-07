#!/usr/bin/env python

"""
This script generates and output a documentation from a list of Makefile files

Example usage:

   python generate_makefile_doc.py Makefile1 Makefile2 ...
"""

import argparse
import re


# Parse a Makefile target line:
#
# Example 1:
# unit: ## @testing Runs the unit tests without coverage reports.
#    name => unit
#    varname => None
#    category => testing
#    doc => Runs the unit tests without coverage reports.
#
# Example 2:
# ${BEAT_NAME}: $(GOFILES_ALL) ## @build build the beat application
#    name => None
#    varname => BEAT_NAME
#    category => testing
#    doc => Runs the unit tests without coverage reports.
regexp_target_doc = re.compile(
    r'^((?P<name>(-|_|\w)+)|(\${(?P<varname>(-|_|\w)+)}))\s*:.*\#\#+\s*@(?P<category>(\w+))\s+(?P<doc>(.*))')


# Parse a Makefile variable assignement:
#
# Example 1:
# BEAT_LICENSE?=ASL 2.0 ## @packaging Software license of the application
#    name => BEAT_LICENSE
#    default => ASL 2.0
#    category => packaging
#    doc => Software license of the application
#
# Example 2:
# BEAT_NAME?=filebeat
#    name => BEAT_NAME
#    default => libbeat
#    category => None
#    doc => None
#
regexp_var_help = re.compile(
    r'^(?P<name>(\w)+)\s*(\?)?=\s*(?P<default>([^\#]+))(\s+\#\#+\s*@(?P<category>(\w+))(:)?\s+(?P<doc>(.*))|\s*$)')


# Parse a Makefile line according to the given regexp
# - insert the dict { name, default, is_variable, category, doc} to the categories dictionary
# - insert the category to the categories_set
# - return a pair [name, value] if the line is a Makefile variable assignement
def parse_line(line, regexp, categories, categories_set):
    matches = regexp.match(line)
    variable = None
    if matches:
        name = None
        variable = False
        try:
            name = matches.group("varname")
            is_variable = True
        except:
            pass
        try:
            default = matches.group("default").strip()
        except:
            default = ""

        if not name:
            name = matches.group("name")
            is_variable = False

        if name:
            variable = [name, default]

        category = matches.group("category")
        if category:
            category = category.replace("_", " ").capitalize()
            doc = matches.group("doc").rstrip('.').rstrip()
            doc = doc[0].capitalize() + doc[1:]  # Capitalize the first word

            if category not in categories_set:
                categories_set.append(category)
                categories[category] = []

            categories[category].append({
                "name": name,
                "doc": doc,
                "is_variable": is_variable,
                "default": default,
            })
    return variable


# Substitute all Makefile targets whose names are Makefile variables by their final name.
#
# Example in Makefile:
#
# ${BEAT_NAME}: $(GOFILES_ALL) ## @build build the beat application
# 	go build
#
# BEAT_NAME is a Makefile target whose name ${BEAT_NAME} is a Makefile variable.
# The name of the rule is changed from "BEAT_NAME" to "filebeat"
#
def substitute_variable_targets(targets, variables):
    target_variables = ([target for category in targets for target in targets[category] if target['is_variable']])
    for variable in target_variables:
        variable['name'] = variables[variable['name']]
        variable['variable'] = False

# Display the help to stdout


def print_help(categories, categories_set):
    column_size = max(len(rule["name"]) for category in categories_set for rule in categories[category])
    for category in categories_set:
        print("\n{}:".format(category))
        for rule in categories[category]:
            if "name" in rule:
                name = rule["name"]
            if "varname" in rule:
                name = rule["varname"]
            default = rule["default"]
            print("\t{target: <{fill}}\t{doc}.{default}".format(
                target=rule["name"], fill=column_size,
                doc=rule["doc"],
                default=(" Default: {}".format(default) if default else "")))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Generate documentation from a list of Makefile files")

    parser.add_argument("--variables", dest='variables',
                        action='store_true')

    parser.add_argument("files", nargs="+", type=argparse.FileType('r'),
                        help="list of Makefiles to analyze",
                        default=None)
    args = parser.parse_args()

    categories_targets = {}
    categories_vars = {}
    categories_targets_set = []
    categories_vars_set = []
    variables = {}

    for file in args.files:
        for line in file.readlines():
            parse_line(line, regexp_target_doc, categories_targets, categories_targets_set)
            variable = parse_line(line, regexp_var_help, categories_vars, categories_vars_set)
            if variable and variable[0] not in variables:
                variables[variable[0]] = variable[1]

    substitute_variable_targets(categories_targets, variables)

    if not args.variables:
        print("Usage: make [target] [VARIABLE=value]")
        print_help(categories_targets, categories_targets_set)
    else:
        print_help(categories_vars, categories_vars_set)
