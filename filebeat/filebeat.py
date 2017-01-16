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

    # load_dashboards(args)
    load_datasets(args, modules)


def load_dashboards(args):
    cmd = ["../libbeat/dashboards/import_dashboards",
           "-dir", "_meta/kibana",
           "-es", args.es]
    subprocess.Popen(cmd).wait()


def load_datasets(args, modules):
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
            load_fileset(args, module, fileset,
                         os.path.join(path, fileset))

    run_filebeat(args)


def load_fileset(args, module, fileset, path):
    manifest = yaml.load(file(os.path.join(path, "manifest.yml"), "r"))
    var = evaluate_vars(args, manifest["var"], module, fileset)
    var["beat"] = dict(module=module, fileset=fileset, path=path, args=args)
    print("Evaluated variables: {}".format(var))

    load_pipeline(var, manifest["ingest_pipeline"])


def evaluate_vars(args, var_in, module, fileset):
    var = {
        "builtin": get_builtin_vars()
    }
    for vals in var_in:
        name = vals["name"]
        var[name] = vals["default"]
        if sys.platform == "darwin" and "os.darwin" in vals:
            var[name] = vals["os.darwin"]
        elif sys.platform == "windows" and "os.windows" in vals:
            var[name] = vals["os.windows"]

        if isinstance(var[name], basestring):
            var[name] = apply_template(var[name], var)
        elif isinstance(var[name], list):
            # only supports array of strings atm
            var[name] = [apply_template(x, var) for x in var[name]]

    return var


def apply_template(tpl, var):
    tpl = tpl.replace("{{.", "{{")     # Go templates
    return Template(tpl).render(var)


def get_builtin_vars():
    host = socket.gethostname()
    hostname, _, domain = host.partition(".")
    # separate the domain
    return {
        "hostname": hostname,
        "domain": domain
    }


def load_pipeline(var, pipeline):
    path = os.path.join(var["beat"]["path"], apply_template(pipeline, var))
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


def run_filebeat(args):
    cfg_template = """
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
            dict(es=args.es)))
        print("Wrote configuration file: {}".format(cfgfile.name))
    os.close(fd)

    cmd = ["./filebeat.test", "-systemTest",
           "-modules", args.modules,
           "-e", "-c", cfgfile.name, "-d", "*"]
    for override in args.M:
        cmd.extend(["-M", override])
    if args.once:
        cmd.extend(["-M", "*.*.prospector.close_eof=true"])
        cmd.append("-once")
    print("Starting filebeat: " + " ".join(cmd))

    subprocess.Popen(cmd).wait()


if __name__ == "__main__":
    sys.exit(main())
