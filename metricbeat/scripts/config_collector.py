import os
import argparse
import yaml

# Collects config for all modules

header_full = """########################## Metricbeat Configuration ###########################

# This file is a full configuration example documenting all non-deprecated
# options in comments. For a shorter configuration example, that contains only
# the most common options, please see metricbeat.yml in the same directory.
#
# You can find the full configuration reference here:
# https://www.elastic.co/guide/en/beats/metricbeat/index.html

#==========================  Modules configuration ============================
metricbeat.modules:

"""

header_short = """###################### Metricbeat Configuration Example #######################

# This file is an example configuration file highlighting only the most common
# options. The metricbeat.full.yml file from the same directory contains all the
# supported options with more comments. You can use it as a reference.
#
# You can find the full configuration reference here:
# https://www.elastic.co/guide/en/beats/metricbeat/index.html

#==========================  Modules configuration ============================
metricbeat.modules:

"""


def collect(beat_path, full=False):

    base_dir = beat_path + "/module"
    path = os.path.abspath(base_dir)

    # yml file

    if full:
        config_yml = header_full
    else:
        config_yml = header_short

    # Read the modules list but put "system" first
    modules = ["system"]
    for module in os.listdir(base_dir):
        if module != "system":
            modules.append(module)

    # Iterate over all modules
    for module in modules:

        beat_path = path + "/" + module + "/_meta"

        module_configs = beat_path + "/config.yml"

        # By default, short config is read if short is set
        short_config = True

        # Check if full config exists
        if full:
            full_module_config = beat_path + "/config.full.yml"
            if os.path.isfile(full_module_config):
                module_configs = full_module_config

        # Only check folders where config exists
        if not os.path.isfile(module_configs):
            continue

        # Load title from fields.yml
        with open(beat_path + "/fields.yml") as f:
            fields = yaml.load(f.read())
            title = fields[0]["title"]

            # Check if short config was disabled in fields.yml
            if not full and "short_config" in fields[0]:
                short_config = fields[0]["short_config"]

        if not full and short_config is False:
            continue

        config_yml += get_title_line(title)

        # Load module yaml
        with file(module_configs) as f:
            for line in f:
                config_yml += line

        config_yml += "\n"
    # output string so it can be concatenated
    print config_yml


# Makes sure every title line is 79 + newline chars long
def get_title_line(title):
    dashes = (79 - 10 - len(title)) / 2

    line = "#"
    line += "-" * dashes
    line += " " + title + " Module "
    line += "-" * dashes

    return line[0:78] + "\n"

if __name__ == "__main__":

    parser = argparse.ArgumentParser(
        description="Collects modules config")
    parser.add_argument("path", help="Path to the beat folder")
    parser.add_argument("--full", action="store_true",
                        help="Collect the full versions")

    args = parser.parse_args()
    beat_path = args.path

    collect(beat_path, args.full)
