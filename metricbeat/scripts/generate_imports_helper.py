from os.path import abspath, isdir, join
from os import listdir

comment = """Package include imports all Module and MetricSet packages so that they register
their factories with the global registry. This package can be imported in the
main package to automatically register all of the standard supported Metricbeat
modules."""

modules_by_platform = [
    {
        "file_suffix": "_docker",
        "build_tags": "// +build linux darwin windows\n\n",
        "modules": ["docker", "kubernetes"],
    },
]


def get_importable_lines(go_beat_path, import_line):
    path = abspath("module")

    imports_by_module = []
    common_lines = []
    modules = [m for m in listdir(path) if isdir(join(path, m)) and m != "_meta"]
    not_common_modules = []
    for m in modules_by_platform:
        not_common_modules.extend(m["modules"])

    for platform_info in modules_by_platform:
        lines = []
        for module in modules:
            module_import = import_line.format(beat_path=go_beat_path, module="module", name=module)
            if module in not_common_modules:
                lines = _collect_imports_from_module(path, module, module_import, go_beat_path, import_line, lines)
            else:
                common_lines = _collect_imports_from_module(
                    path, module, module_import, go_beat_path, import_line, common_lines)

        if lines is not None:
            imports_by_module.append({
                "file_suffix": platform_info["file_suffix"],
                "build_tags": platform_info["build_tags"],
                "imported_lines": lines,
            })

    imports_by_module.append({
        "file_suffix": "_common",
        "build_tags": "",
        "imported_lines": sorted(common_lines),
    })

    return imports_by_module


def _collect_imports_from_module(path, module, module_import, go_beat_path, import_line, imported_lines):
    imported_lines.append(module_import)

    module_path = join(path, module)
    ignore = ["_meta", "vendor", "mtest"]
    metricsets = [m for m in listdir(module_path) if isdir(join(module_path, m)) and m not in ignore]
    for metricset in metricsets:
        metricset_name = "{}/{}".format(module, metricset)
        metricset_import = import_line.format(beat_path=go_beat_path, module="module", name=metricset_name)
        imported_lines.append(metricset_import)

    return sorted(imported_lines)
