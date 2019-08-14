import os
import argparse
import yaml
import six
import glob
import re

# Collects module configs to modules.d

REFERENCE_CONFIG_RE = re.compile('.+\.reference\.yml')


def collect(beat_name, docs_branch):

    base_dir = "module"
    path = os.path.abspath("module")

    # TODO add module release status if beta or experimental
    header = """# Module: {module}
# Docs: https://www.elastic.co/guide/en/beats/{beat_name}/{docs_branch}/{beat_name}-module-{module}.html

"""

    # Create directory for module confs
    os.mkdir(os.path.abspath('modules.d'))

    # Iterate over all modules
    for module in sorted(os.listdir(base_dir)):

        module_confs = path + '/' + module + '/_meta/config*.yml'
        for module_conf in glob.glob(module_confs):

            # Ignore reference confs
            if REFERENCE_CONFIG_RE.match(module_conf):
                continue

            if os.path.isfile(module_conf) == False:
                continue

            module_file = header.format(module=module, beat_name=beat_name, docs_branch=docs_branch)
            disabled_config_filename = os.path.basename(module_conf).replace('config', module) + '.disabled'

            with open(module_conf) as f:
                module_file += f.read()

            # Write disabled module conf
            with open(os.path.abspath('modules.d/') + '/' + disabled_config_filename, 'w') as f:
                f.write(module_file)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Collects modules confs")
    parser.add_argument("--beat", help="Beat name")
    parser.add_argument("--docs_branch", help="Docs branch")

    args = parser.parse_args()
    beat_name = args.beat
    docs_branch = args.docs_branch

    collect(beat_name, docs_branch)
