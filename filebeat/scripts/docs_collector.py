import os
import argparse
import yaml
import six
import glob

# Collects docs for all modules


def collect(beat_name):

    base_dir = "module"
    path = os.path.abspath("module")

    generated_note = """////
This file is generated! See scripts/docs_collector.py
////

"""

    # Include both OSS and Elastic licensed modules.
    modules = []
    modules.extend(glob.glob('module/*'))
    modules.extend(glob.glob('../x-pack/filebeat/module/*'))

    modules_list = {}

    # Iterate over all modules
    for module in sorted(modules):
        module_dir = os.path.abspath(module)
        start = os.path.dirname(os.path.dirname(os.path.dirname(__file__)))
        module_rel_path = os.path.relpath(module,start)
        module = os.path.basename(module)
        module_doc = os.path.join(module_dir, "_meta/docs.asciidoc")

        # Only check folders where docs.asciidoc exists
        if not os.path.isfile(module_doc):
            continue

        module_file = generated_note

        module_file += ":edit_url: https://github.com/elastic/beats/edit/main/" + module_rel_path + "/_meta/docs.asciidoc\n\n"

        module_file += "[[filebeat-module-" + module + "]]\n"

        with open(module_doc, encoding='utf_8') as f:
            module_file += f.read()

        beat_path = os.path.join(module_dir, "_meta")

        # Load title from fields.yml
        with open(beat_path + "/fields.yml", encoding='utf_8') as f:
            fields = yaml.load(f.read(), Loader=yaml.FullLoader)
            title = fields[0]["title"]

        modules_list[module] = title

        module_file += """

[float]
=== Fields

For a description of each field in the module, see the
<<exported-fields-""" + module + """,exported fields>> section.

"""

        # Write module docs
        docs_path = os.path.join(os.path.abspath("docs"), "modules", module + ".asciidoc")
        with open(docs_path, 'w', encoding='utf_8') as f:
            f.write(module_file)

    module_list_output = generated_note
    module_list_output += "  * <<filebeat-modules-overview>>\n"
    for m, title in sorted(six.iteritems(modules_list)):
        module_list_output += "  * <<filebeat-module-" + m + ">>\n"

    module_list_output += "\n\n--\n\n"
    module_list_output += "include::modules-overview.asciidoc[]\n"
    for m, title in sorted(six.iteritems(modules_list)):
        module_list_output += "include::modules/" + m + ".asciidoc[]\n"
    module_list_output += ":edit_url:"
    # Write module link list
    with open(os.path.abspath("docs") + "/modules_list.asciidoc", 'w', encoding='utf_8') as f:
        f.write(module_list_output)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Collects modules docs")
    parser.add_argument("--beat", help="Beat name")

    args = parser.parse_args()
    beat_name = args.beat

    collect(beat_name)
