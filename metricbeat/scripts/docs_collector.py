import argparse
import os
import six
import yaml


# Collects docs for all modules and metricset


def collect(beat_name):
    oss_base_dir = "module"
    oss_path = os.path.abspath(oss_base_dir)
    xpack_base_dir = "../x-pack/metricbeat/module"
    xpack_path = os.path.abspath(xpack_base_dir)

    generated_note = """////
This file is generated! See scripts/docs_collector.py
////

"""

    modules_list = {}

    modules_path = [{
        'base_dir': oss_base_dir,
        'path': oss_path + "/" + module,
        'name': module,
        'metricsets': sorted(os.listdir(oss_path + "/" + module))
    } for module in filter(lambda module: os.path.isfile(oss_path + "/" + module + "/_meta/docs.asciidoc"),
                           os.listdir(oss_base_dir))]

    if os.path.isdir(os.path.abspath(xpack_base_dir)):
        modules_path += [{
            'base_dir': xpack_base_dir,
            'path': xpack_path + "/" + module,
            'name': module,
            'metricsets': sorted(os.listdir(xpack_path + "/" + module)),
        } for module in filter(lambda module: os.path.isfile(xpack_path + "/" + module + "/_meta/docs.asciidoc"),
                               os.listdir(xpack_path))]

    # Iterate over all modules
    for module in sorted(modules_path):
        # Only check folders where docs.asciidoc exists
        if os.path.isfile(module['path'] + "/_meta/docs.asciidoc") == False:
            continue

        # Create directory for each module
        os.mkdir(os.path.abspath("docs") + "/modules/" + module['name'])

        module_file = generated_note
        module_meta_path = module['path'] + "/_meta"

        # Load module fields.yml
        module_fields = ""
        with open(module_meta_path + "/fields.yml") as f:
            module_fields = yaml.load(f.read())
            module_fields = module_fields[0]

        title = module_fields["title"]

        module_file += "[[metricbeat-module-" + module['name'] + "]]\n"

        module_file += "== {} module\n\n".format(title)

        release = get_release(module_fields)
        if release != "ga":
            module_file += "{}[]\n\n".format(release)

        with open(module['path'] + "/_meta/docs.asciidoc") as f:
            module_file += f.read()

        modules_list[module['name']] = {}
        modules_list[module['name']]["title"] = title
        modules_list[module['name']]["release"] = release
        modules_list[module['name']]["dashboards"] = os.path.exists(module_meta_path + "/kibana")
        modules_list[module['name']]["metricsets"] = {}

        config_file = module_meta_path + "/config.reference.yml"

        if os.path.isfile(config_file) == False:
            config_file = module_meta_path + "/config.yml"

        # Add example config file
        if os.path.isfile(config_file) == True:

            module_file += """

[float]
=== Example configuration

The """ + title + """ module supports the standard configuration options that are described
in <<configuration-metricbeat>>. Here is an example configuration:

[source,yaml]
----
""" + beat_name + ".modules:\n"

            # Load metricset yaml
            with open(config_file) as f:
                # Add 2 spaces for indentation in front of each line
                for line in f:
                    module_file += line

            module_file += "----\n\n"

        # HTTP/SSL helpers
        settings = get_settings(module_fields)
        helper_added = False
        if 'ssl' in settings:
            module_file += (
                "This module supports TLS connections when using `ssl`"
                " config field, as described in <<configuration-ssl>>.\n")
            helper_added = True
        if 'http' in settings:
            module_file += "It also supports the options described in <<module-http-config-options>>.\n"
            helper_added = True
        if helper_added:
            module_file += "\n"

        # Add metricsets title as below each metricset adds its link
        module_file += "[float]\n"
        module_file += "=== Metricsets\n\n"
        module_file += "The following metricsets are available:\n\n"

        module_links = ""
        module_includes = ""

        # Iterate over all metricsets
        for metricset in module['metricsets']:

            metricset_meta = metricset + "/_meta"
            metricset_docs = metricset_meta + "/docs.asciidoc"
            metricset_fields_path = module['path'] + "/" + metricset_meta + "/fields.yml"

            # Only check folders where docs.asciidoc exists
            if not os.path.isfile(module['path'] + "/" + metricset_docs):
                continue

            link_name = "metricbeat-metricset-" + module['name'] + "-" + metricset
            link = "<<" + link_name + "," + metricset + ">>"
            reference = "[[" + link_name + "]]"

            modules_list[module['name']]["metricsets"][metricset] = {}
            modules_list[module['name']]["metricsets"][metricset]["title"] = metricset
            modules_list[module['name']]["metricsets"][metricset]["link"] = link

            module_links += "* " + link + "\n\n"

            module_includes += "include::" + module['name'] + "/" + metricset + ".asciidoc[]\n\n"

            metricset_file = generated_note

            # Add reference to metricset file and include file
            metricset_file += reference + "\n"

            with open(metricset_fields_path) as f:
                metricset_fields = yaml.load(f.read())
                metricset_fields = metricset_fields[0]

            # Read local fields.yml
            # create title out of module and metricset set name
            # Add release fag
            metricset_file += "=== {} {} metricset\n\n".format(title, metricset)

            release = get_release(metricset_fields)
            if release != "ga":
                metricset_file += "{}[]\n\n".format(get_release(metricset_fields))

            modules_list[module['name']]["metricsets"][metricset]["release"] = release

            metricset_file += 'include::../../../' + module['base_dir'] + "/" + \
                              module['name'] + '/' + metricset + '/_meta/docs.asciidoc[]' + "\n"

            # TODO: This should point directly to the exported fields of the metricset, not the whole module
            metricset_file += """

==== Fields

For a description of each field in the metricset, see the
<<exported-fields-""" + module['name'] + """,exported fields>> section.

"""

            data_file = module['path'] + "/" + metricset + "/_meta/data.json"

            # Add data.json example json document
            if os.path.isfile(data_file) == True:
                metricset_file += "Here is an example document generated by this metricset:"
                metricset_file += "\n\n"

                metricset_file += "[source,json]\n"
                metricset_file += "----\n"
                metricset_file += 'include::../../../' + module['base_dir'] + "/" + \
                                  module['name'] + "/" + metricset + "/_meta/data.json[]\n"
                metricset_file += "----\n"

            # Write metricset docs
            with open(os.path.abspath("docs") + "/modules/" + module['name'] + "/" + metricset + ".asciidoc", 'w') as f:
                f.write(metricset_file)

        module_file += module_links
        module_file += module_includes

        # Write module docs
        with open(os.path.abspath("docs") + "/modules/" + module['name'] + ".asciidoc", 'w') as f:
            f.write(module_file)

    module_list_output = generated_note

    module_list_output += '[options="header"]\n'
    module_list_output += '|===================================\n'
    module_list_output += '|Modules   |Dashboards   |Metricsets   \n'

    for key, m in sorted(six.iteritems(modules_list)):

        release_label = ""
        if m["release"] != "ga":
            release_label = m["release"] + "[]"

        dashboard_no = "image:./images/icon-no.png[No prebuilt dashboards] "
        dashboard_yes = "image:./images/icon-yes.png[Prebuilt dashboards are available] "
        dashboards = dashboard_yes if m["dashboards"] else dashboard_no

        module_list_output += '|{} {}   |{}   |{}  \n'.format("<<metricbeat-module-" + key + "," + m["title"] + ">> ",
                                                              release_label, dashboards, "")

        # Make sure empty entry row spans over all metricset rows for this module
        module_list_output += '.{}+| .{}+|  '.format(len(m["metricsets"]), len(m["metricsets"]))

        for key, ms in sorted(six.iteritems(m["metricsets"])):

            release_label = ""
            if ms["release"] != "ga":
                release_label = ms["release"] + "[]"

            module_list_output += '|{} {}  \n'.format(ms["link"], release_label)

    module_list_output += '|================================'

    module_list_output += "\n\n--\n\n"
    for key, m in sorted(six.iteritems(modules_list)):
        module_list_output += "include::modules/" + key + ".asciidoc[]\n"

    # Write module link list
    with open(os.path.abspath("docs") + "/modules_list.asciidoc", 'w') as f:
        f.write(module_list_output)


def get_release(fields):
    # Fetch release flag from fields. Default if not set is experimental
    release = "experimental"
    if "release" in fields:
        release = fields["release"]
        if release not in ["experimental", "beta", "ga"]:
            raise Exception("Invalid release config: {}".format(release))

    return release


def get_settings(fields):
    # Get the list of common settings flags
    return fields.get('settings', [])


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Collects modules docs")
    parser.add_argument("--beat", help="Beat name")

    args = parser.parse_args()
    beat_name = args.beat

    collect(beat_name)
