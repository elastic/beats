import os
import argparse

# Creates a new metricset with all the necessary file
# In case the module does not exist, also the module is created


def generate_metricset(base_path, module, metricset):

    generate_module(base_path, module)
    metricset_path = base_path + "/module/" + module + "/" + metricset
    meta_path = metricset_path + "/_beat"

    if os.path.isdir(metricset_path):
        print "MericSet already exists. Skipping creating metricset " + metricset + ".\n"
        return

    os.makedirs(meta_path)

    templates = base_path + "/scripts/module/metricset/"


    content = load_file(templates + "metricset.go.tmpl", module, metricset)
    with open(metricset_path + "/metricset.go", "w") as f:
        f.write(content)

    content = load_file(templates + "fields.yml", module, metricset)
    with open(meta_path + "/fields.yml", "w") as f:
        f.write(content)

    content = load_file(templates + "docs.asciidoc", module, metricset)
    with open(meta_path + "/docs.asciidoc", "w") as f:
        f.write(content)

    content = load_file(templates + "data.json", module, metricset)
    with open(meta_path + "/data.json", "w") as f:
        f.write(content)

    print "Metricset " + metricset + " created.\n"


def generate_module(base_path, module):

    module_path = base_path + "/module/" + module
    meta_path = module_path + "/_beat"

    if os.path.isdir(module_path):
        print "Module already exists. Skipping creating module " + module + ".\n"
        return

    os.makedirs(meta_path)

    templates = base_path + "/scripts/module/"

    content = load_file(templates + "fields.yml", module, "")
    with open(meta_path + "/fields.yml", "w") as f:
        f.write(content)

    content = load_file(templates + "docs.asciidoc", module, "")
    with open(meta_path + "/docs.asciidoc", "w") as f:
        f.write(content)

    content = load_file(templates + "config.yml", module, "")
    with open(meta_path + "/config.yml", "w") as f:
        f.write(content)

    content = load_file(templates + "doc.go.tmpl", module, "")
    with open(module_path + "/doc.go", "w") as f:
        f.write(content)

    print "Module " + module + " created.\n"


def load_file(file, module, metricset):
    content = ""
    with open(file) as f:
        content = f.read()

    return content.replace("{module}", module).replace("{metricset}", metricset)

if __name__ == "__main__":

    parser = argparse.ArgumentParser(
        description="Creates a metricset")
    parser.add_argument("module", help="Module name")
    parser.add_argument("metricset", help="Metricset name")

    args = parser.parse_args()

    path = os.path.abspath("./")
    generate_metricset(path, args.module.lower(), args.metricset.lower())
