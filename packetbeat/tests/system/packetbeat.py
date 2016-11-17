import os
import sys
import subprocess
import json
import jinja2
import shutil

sys.path.append('../../../libbeat/tests/system')

from beat.beat import TestCase
from beat.beat import Proc

TRANS_REQUIRED_FIELDS = ["@timestamp", "type", "status",
                         "beat.name", "beat.hostname", "beat.version"]

FLOWS_REQUIRED_FIELDS = ["@timestamp", "type",
                         "beat.name", "beat.hostname", "beat.version"]


class BaseTest(TestCase):

    @classmethod
    def setUpClass(self):
        self.beat_name = "packetbeat"
        super(BaseTest, self).setUpClass()

    def run_packetbeat(self, pcap,
                       cmd="../../packetbeat.test",
                       config="packetbeat.yml",
                       output="packetbeat.log",
                       extra_args=[],
                       debug_selectors=[],
                       exit_code=0,
                       wait_stop=0):
        """
        Executes packetbeat on an input pcap file.
        Waits for the process to finish before returning to
        the caller.
        """

        args = [cmd]

        args.extend([
            "-e",
            "-I", os.path.join("pcaps", pcap),
            "-c", os.path.join(self.working_dir, config),
            "-t",
            "-systemTest",
            "-test.coverprofile", os.path.join(self.working_dir, "coverage.cov"),
            "-waitstop", str(wait_stop),
        ])

        if extra_args:
            args.extend(extra_args)

        if debug_selectors:
            args.extend(["-d", ",".join(debug_selectors)])

        with open(os.path.join(self.working_dir, output), "wb") as outputfile:
            proc = subprocess.Popen(args,
                                    stdout=outputfile,
                                    stderr=subprocess.STDOUT)
            actual_exit_code = proc.wait()
            assert actual_exit_code == exit_code, "Expected exit code to be %d, but it was %d" % (exit_code, actual_exit_code)
            return actual_exit_code

    def start_packetbeat(self,
                         cmd="../../packetbeat.test",
                         config="packetbeat.yml",
                         output="packetbeat.log",
                         extra_args=[],
                         debug_selectors=[]):
        """
        Starts packetbeat and returns the process handle. The
        caller is responsible for stopping / waiting for the
        Proc instance.
        """
        args = [cmd,
                "-e",
                "-c", os.path.join(self.working_dir, config),
                "-systemTest",
                "-test.coverprofile", os.path.join(self.working_dir, "coverage.cov")
                ]

        if extra_args:
            args.extend(extra_args)

        if debug_selectors:
            args.extend(["-d", ",".join(debug_selectors)])

        proc = Proc(args, os.path.join(self.working_dir, output))
        proc.start()
        return proc

    def read_output(self,
                    output_file="output/packetbeat",
                    types=None,
                    required_fields=None):
        jsons = []
        with open(os.path.join(self.working_dir, output_file), "r") as f:
            for line in f:
                document = self.flatten_object(json.loads(line), self.dict_fields)
                if not types or document["type"] in types:
                    jsons.append(document)
        self.all_have_fields(jsons, required_fields or TRANS_REQUIRED_FIELDS)
        self.all_fields_are_expected(jsons, self.expected_fields)
        return jsons


    def setUp(self):

        self.template_env = jinja2.Environment(
                loader=jinja2.FileSystemLoader("config")
        )

        # create working dir
        self.working_dir = os.path.join(self.build_path + "run", self.id())
        if os.path.exists(self.working_dir):
            shutil.rmtree(self.working_dir)
        os.makedirs(self.working_dir)

        try:
            # update the last_run link
            if os.path.islink(self.build_path + "last_run"):
                os.unlink(self.build_path + "last_run")
            os.symlink(self.build_path + "run/{}".format(self.id()), self.build_path + "last_run")
        except:
            # symlink is best effort and can fail when
            # running tests in parallel
            pass

        self.expected_fields, self.dict_fields = self.load_fields()
