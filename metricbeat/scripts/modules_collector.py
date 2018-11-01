import os
import argparse
import yaml
import six

# Collects module configs to modules.d


def collect(docs_branch, is_x_pack):

    # TODO add module release status if beta or experimental
    header = """# Module: {module}
# Docs: https://www.elastic.co/guide/en/beats/metricbeat/{docs_branch}/metricbeat-module-{module}.html

"""

    modules = get_modules_list(is_x_pack)

    # Iterate over all modules
    for (module, beat_path) in modules:

        module_conf_path = beat_path + '/module/' + module + '/_meta/config.yml'
        if os.path.isfile(module_conf_path) == False:
            continue

        module_file = header.format(module=module, docs_branch=docs_branch)

        with open(module_conf_path) as f:
            module_file += f.read()

        # Write disabled module conf
        with open(os.getcwd() + '/modules.d/' + '/' + module + '.yml.disabled', 'w') as f:
            f.write(module_file)


def get_modules_list(is_x_pack):
    modules = []

    beat_path = os.getcwd()
    if is_x_pack:
        # If x_pack_path is defined, path variable points now to xpack beat folder
        for module in os.listdir(beat_path + "/module"):
            modules.append((module, beat_path))

        # Create directory in x-pack
        os.mkdir(beat_path + '/modules.d')

        # Set the path to OSS folder now
        beat_path = os.path.abspath("../../metricbeat")
    else:
        # Create directory in oss
        os.mkdir(beat_path + '/modules.d')

    for module in os.listdir(beat_path + "/module"):
        modules.append((module, beat_path))

    return sorted(modules, key=lambda t: t[0])


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Collects modules confs")
    parser.add_argument("--docs_branch", help="Docs branch")
    parser.add_argument("--is_x_pack", action="store_true")

    args = parser.parse_args()
    docs_branch = args.docs_branch

    collect(docs_branch, args.is_x_pack)
