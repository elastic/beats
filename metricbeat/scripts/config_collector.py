import os

# Collects config for all modules


def collect():

    base_dir = "module"
    path = os.path.abspath("module")

    # yml file
    config_yml = "metricbeat.modules:\n\n"

    # Iterate over all modules
    for module in os.listdir(base_dir):

        module_configs = path + "/" + module + "/_beat/config.yml"

        # Only check folders where fields.yml exists
        if os.path.isfile(module_configs) == False:
            continue

        # Load module yaml
        with file(module_configs) as f:

            # Add 2 spaces for indentation in front of each line
            for line in f:
                if len(line.strip()) > 0:
                    config_yml += "  " + line
                else:
                    config_yml += line

        config_yml += "\n"
    # output string so it can be concatenated
    print config_yml

if __name__ == "__main__":
    collect()



