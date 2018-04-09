import os
import argparse
import yaml
import six

# Collects docs for all modules and metricset


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
        if os.path.isfile(module_doc) == False:
            continue

        # Create directory for each module
        os.mkdir(os.path.abspath("docs") + "/modules/" + module)

        module_file = generated_note
        module_meta_path = path + "/" + module + "/_meta"

        # Load module fields.yml
        module_fields = ""
        with open(module_meta_path + "/fields.yml") as f:
            module_fields = yaml.load(f.read())
            module_fields = module_fields[0]

        title = module_fields["title"]

        module_file += "[[metricbeat-module-" + module + "]]\n"

        module_file += "== {} module\n\n".format(title)

        release = get_release(module_fields)
        if release != "ga":
            module_file += "{}[]\n\n".format(release)

        with open(module_doc) as f:
            module_file += f.read()

        modules_list[module] = {}
        modules_list[module]["title"] = title
        modules_list[module]["release"] = release
        modules_list[module]["dashboards"] = os.path.exists(module_meta_path + "/kibana")
        modules_list[module]["metricsets"] = {}

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

        # HTTP helper
        if 'ssl' in get_settings(module_fields):
            module_file += "This module supports TLS connection when using `ssl` config field, as described in <<configuration-ssl>>.\n\n"

        # Add metricsets title as below each metricset adds its link
        module_file += "[float]\n"
        module_file += "=== Metricsets\n\n"
        module_file += "The following metricsets are available:\n\n"

        module_links = ""
        module_includes = ""

        # Iterate over all metricsets
        for metricset in sorted(os.listdir(base_dir + "/" + module)):

            metricset_meta = path + "/" + module + "/" + metricset + "/_meta"
            metricset_docs = metricset_meta + "/docs.asciidoc"
            metricset_fields_path = metricset_meta + "/fields.yml"

            # Only check folders where docs.asciidoc exists
            if os.path.isfile(metricset_docs) == False:
                continue

            link_name = "metricbeat-metricset-" + module + "-" + metricset
            link = "<<" + link_name + "," + metricset + ">>"
            reference = "[[" + link_name + "]]"

            modules_list[module]["metricsets"][metricset] = {}
            modules_list[module]["metricsets"][metricset]["title"] = metricset
            modules_list[module]["metricsets"][metricset]["link"] = link

            module_links += "* " + link + "\n\n"

            module_includes += "include::" + module + "/" + metricset + ".asciidoc[]\n\n"

            metricset_file = generated_note

            # Add reference to metricset file and include file
            metricset_file += reference + "\n"

            metricset_fields = ""
            with open(metricset_fields_path) as f:
                metricset_fields = yaml.load(f.read())
                metricset_fields = metricset_fields[0]

            # Read local fields.yml
            # create title out of module and metricset set name
            # Add relase fag
            metricset_file += "=== {} {} metricset\n\n".format(title, metricset)

            release = get_release(metricset_fields)
            if release != "ga":
                metricset_file += "{}[]\n\n".format(get_release(metricset_fields))

            modules_list[module]["metricsets"][metricset]["release"] = release

            metricset_file += 'include::../../../module/' + module + '/' + metricset + '/_meta/docs.asciidoc[]' + "\n"

            # TODO: This should point directly to the exported fields of the metricset, not the whole module
            metricset_file += """

==== Fields

For a description of each field in the metricset, see the
<<exported-fields-""" + module + """,exported fields>> section.

"""

            data_file = path + "/" + module + "/" + metricset + "/_meta/data.json"

            # Add data.json example json document
            if os.path.isfile(data_file) == True:
                metricset_file += "Here is an example document generated by this metricset:"
                metricset_file += "\n\n"

                metricset_file += "[source,json]\n"
                metricset_file += "----\n"
                metricset_file += "include::../../../module/" + module + "/" + metricset + "/_meta/data.json[]\n"
                metricset_file += "----\n"

            # Write metricset docs
            with open(os.path.abspath("docs") + "/modules/" + module + "/" + metricset + ".asciidoc", 'w') as f:
                f.write(metricset_file)

        module_file += module_links
        module_file += module_includes

        # Write module docs
        with open(os.path.abspath("docs") + "/modules/" + module + ".asciidoc", 'w') as f:
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
