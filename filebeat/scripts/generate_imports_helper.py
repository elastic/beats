from os.path import abspath, isdir, join
from os import listdir


comment = """Package include imports all input packages so that they register
their factories with the global registry. This package can be imported in the
main package to automatically register all of the standard supported inputs
modules."""


def get_importable_lines(go_beat_path, import_line):
    path = abspath("input")

    imported_prospector_lines = []

    # Skip the file folder, its not an input but I will do the move with another PR
    prospectors = [p for p in listdir(path) if isdir(join(path, p)) and p.find("file") is -1]
    for prospector in sorted(prospectors):
        prospector_import = import_line.format(beat_path=go_beat_path, module="input", name=prospector)
        imported_prospector_lines.append(prospector_import)

    return imported_prospector_lines
