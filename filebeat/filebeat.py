#!/usr/bin/env python
import argparse
import sys
import os
import yaml
import requests
import tempfile
import subprocess
import socket
from jinja2 import Template


def main():
    parser = argparse.ArgumentParser(
        description="PROTOTYPE: start filebeat with a module configuration")
    parser.add_argument("--module", default="",
                        help="From branch")
    parser.add_argument("--nginx", action="store_true",
                        help="Shortcut for --module nginx")
    parser.add_argument("--es", default="http://localhost:9200",
                        help="Elasticsearch URL")
    parser.add_argument("-E", nargs="*", type=str, default=None,
                        help="Variables overrides. e.g. path=/test")

    args = parser.parse_args()
    print args

    if args.nginx:
        args.module = "nginx"

    if args.module == "":
        print("You need to specify a module")
        sys.exit(1)

    load_dashboards(args)
    load_datasets(args, args.module)


def load_dashboards(args):
    cmd = ["../libbeat/dashboards/import_dashboards",
           "-dir", "_meta/kibana",
           "-es", args.es]
    subprocess.Popen(cmd).wait()


def load_datasets(args, module):
    path = os.path.join("module", module)
    if not os.path.isdir(path):
        print("Module {} not found".format(module))
        sys.exit(1)
    print("Found module {} in {}".format(module, path))

    filesets = [name for name in os.listdir(path) if
                os.path.isfile(os.path.join(path, name, "manifest.yml"))]

    print("Found filesets: {}".format(filesets))

    prospectors = ""
    for fileset in filesets:
        prospectors += load_fileset(args, module, fileset,
                                    os.path.join(path, fileset))

    run_filebeat(args, prospectors)

    print("Generated configuration: {}".format(prospectors))


def load_fileset(args, module, fileset, path):
    manifest = yaml.load(file(os.path.join(path, "manifest.yml"), "r"))
    var = evaluate_vars(args, manifest["vars"])
    var["beat"] = dict(module=module, fileset=fileset, path=path, args=args)
    print("Evaluated variables: {}".format(var))

    load_pipeline(var, manifest["ingest_pipeline"])
    generate_prospectors(var, manifest["prospectors"])

    return var["beat"]["prospectors"]


def evaluate_vars(args, var_in):
    var = {
        "builtin": get_builtin_vars()
    }
    for name, vals in var_in.items():
        var[name] = vals["default"]

        if sys.platform == "darwin" and "os.darwin" in vals:
            var[name] = vals["os.darwin"]
        elif sys.platform == "windows" and "os.windows" in vals:
            var[name] = vals["os.windows"]

        var[name] = Template(var[name]).render(var)

    # overrides
    if args.E is not None:
        for pair in args.E:
            key, val = pair.partition("=")[::2]
            var[key] = val

    return var


def get_builtin_vars():
    host = socket.gethostname()
    hostname, _, domain = host.partition(".")
    # separate the domain
    return {
        "hostname": hostname,
        "domain": domain
    }


def load_pipeline(var, pipeline):
    path = os.path.join(var["beat"]["path"], Template(pipeline).render(var))
    print("Loading ingest pipeline: {}".format(path))
    var["beat"]["pipeline_id"] = var["beat"]["module"] + '-' + var["beat"]["fileset"] + \
        '-' + os.path.splitext(os.path.basename(path))[0]
    print("Pipeline id: {}".format(var["beat"]["pipeline_id"]))

    with open(path, "r") as f:
        contents = f.read()

    r = requests.put("{}/_ingest/pipeline/{}"
                     .format(var["beat"]["args"].es,
                             var["beat"]["pipeline_id"]),
                     data=contents)
    if r.status_code >= 300:
        print("Error posting pipeline: {}".format(r.text))
        sys.exit(1)


def run_filebeat(args, prospectors):
    cfg_template = """
filebeat.prospectors:
{{prospectors}}

output.elasticsearch.hosts: ["{{es}}"]
output.elasticsearch.pipeline: "%{[fields.pipeline_id]}"
"""
    fd, fname = tempfile.mkstemp(suffix=".yml", prefix="filebeat-",
                                 text=True)
    with open(fname, "w") as cfgfile:
        cfgfile.write(Template(cfg_template).render(
            dict(prospectors=prospectors, es=args.es)))
        print("Wrote configuration file: {}".format(cfgfile.name))
    os.close(fd)

    cmd = ["./filebeat", "-e", "-c", cfgfile.name, "-d", "*"]

    subprocess.Popen(cmd).wait()


def generate_prospectors(var, prospectors):
    var["beat"]["prospectors"] = ""
    for pr in prospectors:
        path = os.path.join(var["beat"]["path"], Template(pr).render(var))
        with open(path, "r") as f:
            contents = Template(f.read()).render(var)
        var["beat"]["prospectors"] += "\n" + contents


if __name__ == "__main__":
    sys.exit(main())
