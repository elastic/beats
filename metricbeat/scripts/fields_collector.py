import os

# Collects fields for all modules and metricset


def collect():

    base_dir = "module"
    path = os.path.abspath("module")

    # yml file
    fields_yml = ""

    # Iterate over all modules
    for module in sorted(os.listdir(base_dir)):

        module_fields = path + "/" + module + "/_meta/fields.yml"

        # Only check folders where fields.yml exists
        if os.path.isfile(module_fields) == False:
            continue

        # Load module yaml
        with file(module_fields) as f:
            tmp = f.read()
            fields_yml += tmp


        # Iterate over all metricsets
        for metricset in sorted(os.listdir(base_dir + "/" + module)):

            metricset_fields = path + "/" + module + "/" + metricset + "/_meta/fields.yml"

            # Only check folders where fields.yml exists
            if os.path.isfile(metricset_fields) == False:
                continue

            # Load metricset yaml
            with file(metricset_fields) as f:
                # Add 4 spaces for indentation in front of each line
                for line in f:
                    if len(line.strip()) > 0:
                        fields_yml += "    " + "    " + line
                    else:
                        fields_yml += line

            # Add newline to make sure indentation is correct
            fields_yml += "\n"

    # output string so it can be concatenated
    print fields_yml

if __name__ == "__main__":
    collect()



