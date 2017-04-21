import os
import argparse
import yaml

# Collects docs for all modules


def collect(beat_name):

    base_dir = "module"
    path = os.path.abspath("module")

    generated_note = """////
This file is generated! See scripts/docs_collector.py
////

"""

    modules_list = {}

    # Iterate over all modules
    for module in sorted(os.listdir(base_dir)):

        module_doc = path + "/" + module + "/_meta/docs.asciidoc"

        # Only check folders where docs.asciidoc exists
        if not os.path.isfile(module_doc):
            continue

        module_file = generated_note
        module_file += "[[filebeat-module-" + module + "]]\n"

        with file(module_doc) as f:
            module_file += f.read()

        beat_path = path + "/" + module + "/_meta"

        # Load title from fields.yml
        with open(beat_path + "/fields.yml") as f:
            fields = yaml.load(f.read())
            title = fields[0]["title"]

        modules_list[module] = title

        module_file += """

[float]
=== Fields

For a description of each field in the metricset, see the
<<exported-fields-""" + module + """,exported fields>> section.

"""

        # Write module docs
        with open(os.path.abspath("docs") + "/modules/" +
                  module + ".asciidoc", 'w') as f:
            f.write(module_file)

    module_list_output = generated_note
    module_list_output += "  * <<filebeat-modules-overview>>\n"
    for m, title in sorted(modules_list.iteritems()):
        module_list_output += "  * <<filebeat-module-" + m + ">>\n"
    module_list_output += "  * <<filebeat-modules-devguide>>\n"

    module_list_output += "\n\n--\n\n"
    module_list_output += "include::modules-overview.asciidoc[]\n"
    for m, title in sorted(modules_list.iteritems()):
        module_list_output += "include::modules/" + m + ".asciidoc[]\n"
    module_list_output += "include::modules-dev-guide.asciidoc[]\n"

    # Write module link list
    with open(os.path.abspath("docs") + "/modules_list.asciidoc", 'w') as f:
        f.write(module_list_output)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Collects modules docs")
    parser.add_argument("--beat", help="Beat name")

    args = parser.parse_args()
    beat_name = args.beat

    collect(beat_name)
