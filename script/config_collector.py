import argparse
import os
import yaml


# Collects config for all modules

# yml file
config_yml = "\n#==========================  Modules configuration ============================\n"


def collect(oss_beat_path, x_pack_path, full=False):

    # Read the modules list but put "system" first
    modules = [("system", os.path.abspath(oss_beat_path) + "/module/system/_meta")]
    modules_temp = get_modules_from_beat(oss_beat_path)

    # When called form a x-pack beat, a flag with the path is provided
    if x_pack_path is not None:
        modules_temp += get_modules_from_beat(x_pack_path)

    for (module, path) in sorted(modules_temp, key=lambda tup: tup[0]):
        if module != "system":
            modules.append((module, path))

    write_modules(modules, full)


def get_modules_from_beat(beat_path):
    modules = []

    base_dir = beat_path + "/module"
    path = os.path.abspath(base_dir)

    for module in os.listdir(base_dir):
        if module != "system":
            modules.append((module, path + "/" + module + "/_meta"))

    return modules


def write_modules(modules, full):    # Iterate over all modules
    global config_yml

    for (module, beat_path) in modules:

        # beat_path = path + "/" + module + "/_meta"

        module_configs = beat_path + "/config.yml"

        # By default, short config is read if short is set
        short_config = False

        # Check if full config exists
        if full:
            full_module_config = beat_path + "/config.reference.yml"
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
        with open(module_configs) as f:
            for line in f:
                config_yml += line

        config_yml += "\n"
    # output string so it can be concatenated
    print(config_yml)


# Makes sure every title line is 79 + newline chars long
def get_title_line(title):
    dashes = (79 - 10 - len(title)) // 2

    line = "#"
    line += "-" * dashes
    line += " " + title + " Module "
    line += "-" * dashes

    return line[0:78] + "\n"


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Collects modules config")
    parser.add_argument("path", help="Path to the beat folder")
    parser.add_argument("--beat", help="Beat name")
    parser.add_argument("--x_pack_path", help="Path to x-pack folder with the beat")
    parser.add_argument("--full", action="store_true",
                        help="Collect the full versions")

    args = parser.parse_args()
    beat_name = args.beat
    beat_path = args.path

    config_yml += beat_name + """.modules:

"""

    collect(beat_path, args.x_pack_path, args.full)
