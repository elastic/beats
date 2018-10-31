from os.path import abspath, isdir, join
from os import listdir

comment = """Package include imports all Module and MetricSet packages so that they register
their factories with the global registry. This package can be imported in the
main package to automatically register all of the standard supported Metricbeat
modules."""


def get_importable_lines(go_beat_path, import_line):
    path = abspath("module")
    return get_importable_lines_with_module_path(go_beat_path, import_line, path)

def get_importable_lines_with_module_path(go_beat_path, import_line, beat_root_module_path):
    imported_lines = []
    modules = [m for m in listdir(beat_root_module_path) if isdir(join(beat_root_module_path, m)) and m != "_meta"]
    for module in modules:
        module_import = import_line.format(beat_path=go_beat_path, module="module", name=module)
        imported_lines.append(module_import)

        module_path = join(beat_root_module_path, module)
        ignore = ["_meta", "vendor", "mtest"]
        metricsets = [m for m in listdir(module_path) if isdir(join(module_path, m)) and m not in ignore]
        for metricset in metricsets:
            metricset_name = "{}/{}".format(module, metricset)
            metricset_import = import_line.format(beat_path=go_beat_path, module="module", name=metricset_name)
            imported_lines.append(metricset_import)

    return sorted(imported_lines)
