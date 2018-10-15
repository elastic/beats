import os
import argparse
import yaml
import six

# Collects module configs to modules.d


def collect(docs_branch):

    base_dir = "module"
    path = os.path.abspath("module")

    # TODO add module release status if beta or experimental
    header = """# Module: {module}
# Docs: https://www.elastic.co/guide/en/beats/metricbeat/{docs_branch}/metricbeat-module-{module}.html

"""

    # Create directory for module confs
    os.mkdir(os.path.abspath('modules.d'))

    # Iterate over all modules
    for module in sorted(os.listdir(base_dir)):

        module_conf = path + '/' + module + '/_meta/config.yml'
        if os.path.isfile(module_conf) == False:
            continue

        module_file = header.format(module=module, docs_branch=docs_branch)

        with open(module_conf) as f:
            module_file += f.read()

        # Write disabled module conf
        with open(os.path.abspath('modules.d/') + '/' + module + '.yml.disabled', 'w') as f:
            f.write(module_file)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Collects modules confs")
    parser.add_argument("--docs_branch", help="Docs branch")

    args = parser.parse_args()
    docs_branch = args.docs_branch

    collect(docs_branch)
