import os

# Collects config for all modules

header = """###################### Metricbeat Configuration Example #######################

#==========================  Modules configuration ============================
metricbeat.modules:

"""


def collect():

    base_dir = "module"
    path = os.path.abspath("module")

    # yml file
    config_yml = header

    # Iterate over all modules
    for module in os.listdir(base_dir):

        module_configs = path + "/" + module + "/_beat/config.yml"

        # Only check folders where fields.yml exists
        if not os.path.isfile(module_configs):
            continue

        # Load module yaml
        with file(module_configs) as f:
            for line in f:
                config_yml += line

        config_yml += "\n"
    # output string so it can be concatenated
    print config_yml

if __name__ == "__main__":
    collect()
