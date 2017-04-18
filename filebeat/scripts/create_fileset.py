import os
import argparse

# Creates a new fileset with all the necessary files.
# In case the module does not exist, also the module is created.


def generate_fileset(base_path, metricbeat_path, module, fileset):

    generate_module(base_path, metricbeat_path, module, fileset)
    fileset_path = base_path + "/module/" + module + "/" + fileset
    meta_path = fileset_path + "/_meta"

    if os.path.isdir(fileset_path):
        print("Fileset already exists. Skipping creating fileset {}"
              .format(fileset))
        return

    os.makedirs(meta_path)
    os.makedirs(os.path.join(fileset_path, "test"))

    templates = metricbeat_path + "/scripts/module/fileset/"

    content = load_file(templates + "fields.yml", module, fileset)
    with open(meta_path + "/fields.yml", "w") as f:
        f.write(content)

    os.makedirs(os.path.join(fileset_path, "config"))
    content = load_file(templates + "/config/config.yml", module, fileset)
    with open("{}/config/{}.yml".format(fileset_path, fileset), "w") as f:
        f.write(content)

    os.makedirs(os.path.join(fileset_path, "ingest"))
    content = load_file(templates + "/ingest/pipeline.json", module, fileset)
    with open("{}/ingest/pipeline.json".format(fileset_path), "w") as f:
        f.write(content)

    content = load_file(templates + "/manifest.yml", module, fileset)
    with open("{}/manifest.yml".format(fileset_path), "w") as f:
        f.write(content)

    print("Fileset {} created.".format(fileset))


def generate_module(base_path, metricbeat_path, module, fileset):

    module_path = base_path + "/module/" + module
    meta_path = module_path + "/_meta"

    if os.path.isdir(module_path):
        print("Module already exists. Skipping creating module {}"
              .format(module))
        return

    os.makedirs(meta_path)

    templates = metricbeat_path + "/scripts/module/"

    content = load_file(templates + "fields.yml", module, "")
    with open(meta_path + "/fields.yml", "w") as f:
        f.write(content)

    content = load_file(templates + "docs.asciidoc", module, "")
    with open(meta_path + "/docs.asciidoc", "w") as f:
        f.write(content)

    print("Module {} created.".format(module))


def load_file(file, module, fileset):
    content = ""
    with open(file) as f:
        content = f.read()

    return content.replace("{module}", module).replace("{fileset}", fileset)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Creates a fileset")
    parser.add_argument("--module", help="Module name")
    parser.add_argument("--fileset", help="Fileset name")

    parser.add_argument("--path", help="Beat path")
    parser.add_argument("--es_beats",
                        help="The path to the general beats folder")

    args = parser.parse_args()

    if args.path is None:
        args.path = './'
        print("Set default path for beat path: " + args.path)

    if args.es_beats is None:
        args.es_beats = '../'
        print("Set default path for es_beats path: " + args.es_beats)

    if args.module is None or args.module == '':
        args.module = raw_input("Module name: ")

    if args.fileset is None or args.fileset == '':
        args.fileset = raw_input("Fileset name: ")

    path = os.path.abspath(args.path)
    filebeat_path = os.path.abspath(args.es_beats + "/filebeat")

    generate_fileset(path, filebeat_path, args.module.lower(),
                     args.fileset.lower())
