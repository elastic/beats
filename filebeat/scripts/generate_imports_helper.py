from os.path import abspath, isdir, join
from os import listdir


comment = """Package include imports all input packages so that they register
their factories with the global registry. This package can be imported in the
main package to automatically register all of the standard supported inputs
modules."""


def get_importable_lines(go_beat_path, import_line):
    path = abspath("input")

    imported_input_lines = []

    # Skip the file folder, its not an input but I will do the move with another PR
    inputs = [p for p in listdir(path) if isdir(join(path, p)) and p.find("file") is -1]
    for input in sorted(inputs):
        input_import = import_line.format(beat_path=go_beat_path, module="input", name=input)
        imported_input_lines.append(input_import)

    return imported_input_lines
