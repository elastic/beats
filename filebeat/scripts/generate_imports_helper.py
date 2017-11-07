from os.path import abspath, isdir, join
from os import listdir


comment = """Package include imports all prospector packages so that they register
their factories with the global registry. This package can be imported in the
main package to automatically register all of the standard supported prospectors
modules."""


def get_importable_lines(go_beat_path, import_line):
    path = abspath("prospector")

    imported_prospector_lines = []
    prospectors = [p for p in listdir(path) if isdir(join(path, p))]
    for prospector in sorted(prospectors):
        prospector_import = import_line.format(beat_path=go_beat_path, module="prospector", name=prospector)
        imported_prospector_lines.append(prospector_import)

    return imported_prospector_lines
