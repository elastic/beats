from os.path import abspath, isdir, join
from os import listdir


comment = """Package include imports all protos packages so that they register with the global
registry. This package can be imported in the main package to automatically register
all of the standard supported Packetbeat protocols."""


def get_importable_lines(go_beat_path, import_line):
    path = abspath("protos")

    imported_protocol_lines = []
    protocols = [p for p in listdir(path) if isdir(join(path, p))]
    for protocol in sorted(protocols):
        proto_import = import_line.format(beat_path=go_beat_path, module="protos", name=protocol)
        imported_protocol_lines.append(proto_import)

    return imported_protocol_lines
