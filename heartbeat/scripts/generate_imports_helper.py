comment = """Package defaults imports all Monitor packages so that they
register with the global monitor registry. This package can be imported in the
main package to automatically register all of the standard supported Heartbeat
modules."""

from os.path import abspath, isdir, join
from os import listdir


blacklist = [
    "monitors/active/dialchain"
]


def get_importable_lines(go_beat_path, import_line):
    def format(package, name):
        return import_line.format(
            beat_path=go_beat_path,
            module=package,
            name=name)

    def imports(mode):
        package = "monitors/{}".format(mode)
        return [format(package, m) for m in collect_monitors(package)]

    return sorted(imports("active") + imports("passive"))


def collect_monitors(package):
    path = abspath(package)
    if not isdir(path):
        return []
    return [m for m in listdir(path) if is_monitor(package, m)]


def is_monitor(package, name):
    return (name != "_meta" and
            isdir(join(abspath(package), name)) and
            "{}/{}".format(package, name) not in blacklist)
