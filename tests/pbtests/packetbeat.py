import subprocess
import jinja2
import unittest
import os
import shutil
import json


class TestCase(unittest.TestCase):

    def run_packetbeat(self, pcap,
                       cmd="../packetbeat/packetbeat",
                       config="packetbeat.conf",
                       output="packetbeat.log",
                       extra_args=[],
                       debug_selectors=[]):

        args = [cmd]

        args.extend(["-e",
                     "-I", os.path.join("pcaps", pcap),
                     "-c", os.path.join(self.working_dir, config),
                     "-t"])
        if extra_args:
            args.extend(extra_args)

        if debug_selectors:
            args.extend(["-d", ",".join(debug_selectors)])

        with open(os.path.join(self.working_dir, output), "wb") as outputfile:
            proc = subprocess.Popen(args,
                                    stdout=outputfile,
                                    stderr=subprocess.STDOUT)
            proc.wait()

    def render_config_template(self, template="packetbeat.conf.j2",
                               output="packetbeat.conf", **kargs):
        template = self.template_env.get_template(template)
        kargs["pb"] = self
        output_str = template.render(**kargs)
        with open(os.path.join(self.working_dir, output), "wb") as f:
            f.write(output_str)

    def read_output(self, output_file="output/packetbeat"):
        jsons = []
        with open(os.path.join(self.working_dir, output_file), "r") as f:
            for line in f:
                jsons.append(json.loads(line))
        return jsons

    def setUp(self):

        self.template_env = jinja2.Environment(
            loader=jinja2.FileSystemLoader("templates")
        )

        # create working dir
        self.working_dir = os.path.join("run", self.id())
        if os.path.exists(self.working_dir):
            shutil.rmtree(self.working_dir)
        os.makedirs(self.working_dir)

        # update the last_run link
        if os.path.islink("last_run"):
            os.unlink("last_run")
        os.symlink("run/{}".format(self.id()), "last_run")
