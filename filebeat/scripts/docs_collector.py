import os
import argparse
import yaml
import six
import glob

# Collects docs for all modules


def collect(beat_name):
    generated_note = ""

    # Include both OSS and Elastic licensed modules.
    modules = []
    modules.extend(glob.glob('module/*'))
    modules.extend(glob.glob('../x-pack/filebeat/module/*'))

    modules_list = {}

    # Iterate over all modules
    for module in sorted(modules):
        module_dir = os.path.abspath(module)
        module = os.path.basename(module)
        module_doc = os.path.join(module_dir, "_meta/docs.md")

        # Only check folders where docs.md exists
        if not os.path.isfile(module_doc):
            continue

        beat_path = os.path.join(module_dir, "_meta")

        # Load title from fields.yml
        with open(beat_path + "/fields.yml", encoding='utf_8') as f:
            fields = yaml.load(f.read(), Loader=yaml.FullLoader)
            title = fields[0]["title"]

        module_file = generated_note

        module_file += """---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-module-{}.html
---

% This file is generated! See scripts/docs_collector.py

# {} module [filebeat-module-{}]

""".format(module, title, module)

        with open(module_doc, encoding='utf_8') as f:
            module_file += f.read()

        modules_list[module] = title

        module_file += """
## Fields [_fields]

For a description of each field in the module, see the [exported fields](/reference/filebeat/exported-fields-{}.md) section.
""".format(module)

        # Write module docs
        docs_path = os.path.join(os.path.abspath("../docs"), "reference/filebeat",
                                 "filebeat-module-{}.md".format(module))
        with open(docs_path, 'w', encoding='utf_8') as f:
            f.write(module_file)

    module_list_output = """---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-modules.html
---

# Modules [filebeat-modules]

This section contains an [overview](/reference/filebeat/filebeat-modules-overview.md) of the Filebeat modules feature as well as details about each of the currently supported modules.

Filebeat modules require Elasticsearch 5.2 or later.

::::{note}
While {{filebeat}} modules are still supported, we recommend {{agent}} integrations over {{filebeat}} modules. Integrations provide a streamlined way to connect data from a variety of vendors to the {{stack}}. Refer to the [full list of integrations](https://www.elastic.co/integrations/data-integrations). For more information, please refer to the [{{beats}} vs {{agent}} comparison documentation](docs-content://reference/fleet/index.md).
::::


* [*Modules overview*](/reference/filebeat/filebeat-modules-overview.md)
"""

    for m, title in sorted(six.iteritems(modules_list)):
        module_list_output += "* [*{} module*](/reference/filebeat/filebeat-module-{}.md)\n".format(title, m)

    module_list_output += "\n"

    # Write module link list
    with open(os.path.join(os.path.abspath("../docs"), "reference/filebeat", "filebeat-modules.md"), 'w', encoding='utf_8') as f:
        f.write(module_list_output)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Collects modules docs")
    parser.add_argument("--beat", help="Beat name")

    args = parser.parse_args()
    beat_name = args.beat

    collect(beat_name)
