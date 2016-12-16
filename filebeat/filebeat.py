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
    parser.add_argument("--modules", default="",
                        help="From branch")
    parser.add_argument("--es", default="http://localhost:9200",
                        help="Elasticsearch URL")
    parser.add_argument("--index", default=None,
                        help="Elasticsearch index")
    parser.add_argument("--registry", default=None,
                        help="Registry file to use")
    parser.add_argument("-M", nargs="*", type=str, default=None,
                        help="Variables overrides. e.g. path=/test")
    parser.add_argument("--once", action="store_true",
                        help="Run filebeat with the -once flag")

    args = parser.parse_args()
    print args

    # changing directory because we use paths relative to the binary
    os.chdir(os.path.dirname(sys.argv[0]))

    modules = args.modules.split(",")
    if len(modules) == 0:
        print("You need to specify at least a module")
        sys.exit(1)

    load_dashboards(args)
    load_datasets(args, modules)


def load_dashboards(args):
    cmd = ["../libbeat/dashboards/import_dashboards",
           "-dir", "_meta/kibana",
           "-es", args.es]
    subprocess.Popen(cmd).wait()


def load_datasets(args, modules):
    prospectors = ""
    for module in modules:
        path = os.path.join("module", module)
        if not os.path.isdir(path):
            print("Module {} not found".format(module))
            sys.exit(1)
        print("Found module {} in {}".format(module, path))

        filesets = [name for name in os.listdir(path) if
                    os.path.isfile(os.path.join(path, name, "manifest.yml"))]

        print("Found filesets: {}".format(filesets))

        for fileset in filesets:
            prospectors += load_fileset(args, module, fileset,
                                        os.path.join(path, fileset))

    print("Generated configuration: {}".format(prospectors))
    run_filebeat(args, prospectors)


def load_fileset(args, module, fileset, path):
    manifest = yaml.load(file(os.path.join(path, "manifest.yml"), "r"))
    var = evaluate_vars(args, manifest["vars"], module, fileset)
    var["beat"] = dict(module=module, fileset=fileset, path=path, args=args)
    print("Evaluated variables: {}".format(var))

    load_pipeline(var, manifest["ingest_pipeline"])
    generate_prospectors(var, manifest["prospectors"])

    return var["beat"]["prospectors"]


def evaluate_vars(args, var_in, module, fileset):
    var = {
        "builtin": get_builtin_vars()
    }
    for name, vals in var_in.items():
        var[name] = vals["default"]
        if sys.platform == "darwin" and "os.darwin" in vals:
            var[name] = vals["os.darwin"]
        elif sys.platform == "windows" and "os.windows" in vals:
            var[name] = vals["os.windows"]

        if isinstance(var[name], basestring):
            var[name] = Template(var[name]).render(var)
        elif isinstance(var[name], list):
            # only supports array of strings atm
            var[name] = [Template(x).render(var) for x in var[name]]

    # overrides
    if args.M is not None:
        for pair in args.M:
            key, val = pair.partition("=")[::2]
            if key.startswith("{}.{}.".format(module, fileset)):
                key = key[len("{}.{}.".format(module, fileset)):]

                # this is a hack in the prototype only, because
                # here we don't know the type of each variable type.
                if key == "paths":
                    val = val.split(",")
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
    if args.index:
        cfg_template += "\noutput.elasticsearch.index: {}".format(args.index)

    if args.once:
        cfg_template += "\nfilebeat.idle_timeout: 0.5s"

    if args.registry:
        cfg_template += "\nfilebeat.registry_file: {}".format(args.registry)

    fd, fname = tempfile.mkstemp(suffix=".yml", prefix="filebeat-",
                                 text=True)
    with open(fname, "w") as cfgfile:
        cfgfile.write(Template(cfg_template).render(
            dict(prospectors=prospectors, es=args.es)))
        print("Wrote configuration file: {}".format(cfgfile.name))
    os.close(fd)

    cmd = ["./filebeat.test", "-systemTest",
           "-e", "-c", cfgfile.name, "-d", "*"]
    if args.once:
        cmd.append("-once")
    print("Starting filebeat: " + " ".join(cmd))

    subprocess.Popen(cmd).wait()


def generate_prospectors(var, prospectors):
    var["beat"]["prospectors"] = ""
    for pr in prospectors:
        path = os.path.join(var["beat"]["path"], Template(pr).render(var))
        with open(path, "r") as f:
            contents = Template(f.read()).render(var)
        if var["beat"]["args"].once:
            contents += "\n  close_eof: true"
            contents += "\n  scan_frequency: 0.2s"
            if "multiline" in contents:
                contents += "\n  multiline.timeout: 0.2s"

        var["beat"]["prospectors"] += "\n" + contents


if __name__ == "__main__":
    sys.exit(main())
