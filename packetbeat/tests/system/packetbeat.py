import os
import sys
import subprocess
import json

sys.path.append(os.path.join(os.path.dirname(__file__), '../../../libbeat/tests/system'))

from beat.beat import TestCase
from beat.beat import Proc

TRANS_REQUIRED_FIELDS = ["@timestamp", "type", "status",
                         "agent.type", "agent.hostname", "agent.version",
                         "event.kind", "event.category", "event.dataset", "event.start",
                         "source.ip", "destination.ip",
                         "client.ip", "server.ip",
                         "network.type", "network.transport", "network.community_id",
                         ]

FLOWS_REQUIRED_FIELDS = ["@timestamp", "type",
                         "agent.type", "agent.hostname", "agent.version",
                         "event.kind", "event.category", "event.dataset", "event.action", "event.start", "event.end", "event.duration",
                         "source.ip", "destination.ip",
                         "flow.id",
                         "network.type", "network.transport", "network.community_id",
                         ]


class BaseTest(TestCase):

    @classmethod
    def setUpClass(self):
        self.beat_name = "packetbeat"
        self.beat_path = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../"))
        super(BaseTest, self).setUpClass()

    def run_packetbeat(self, pcap,
                       cmd=None,
                       config="packetbeat.yml",
                       output="packetbeat.log",
                       extra_args=[],
                       debug_selectors=[],
                       exit_code=0,
                       real_time=False):
        """
        Executes packetbeat on an input pcap file.
        Waits for the process to finish before returning to
        the caller.
        """

        if cmd is None:
            cmd = self.beat_path + "/packetbeat.test"

        args = [cmd]

        if not real_time:
            args.extend(["-t"])

        args.extend([
            "-e",
            "-I", os.path.join(self.beat_path + "/tests/system/pcaps", pcap),
            "-c", os.path.join(self.working_dir, config),
            "-systemTest",
        ])
        if os.getenv("TEST_COVERAGE") == "true":
            args += [
                "-test.coverprofile", os.path.join(self.working_dir, "coverage.cov"),
            ]

        if extra_args:
            args.extend(extra_args)

        if debug_selectors:
            args.extend(["-d", ",".join(debug_selectors)])

        with open(os.path.join(self.working_dir, output), "wb") as outputfile:
            proc = subprocess.Popen(args,
                                    stdout=outputfile,
                                    stderr=subprocess.STDOUT)
            actual_exit_code = proc.wait()

        if actual_exit_code != exit_code:
            print("============ Log Output =====================")
            with open(os.path.join(self.working_dir, output)) as f:
                print(f.read())
            print("============ Log End Output =====================")
        assert actual_exit_code == exit_code, "Expected exit code to be %d, but it was %d" % (
            exit_code, actual_exit_code)
        return actual_exit_code

    def start_packetbeat(self,
                         cmd=None,
                         config="packetbeat.yml",
                         output="packetbeat.log",
                         extra_args=[],
                         debug_selectors=[]):
        """
        Starts packetbeat and returns the process handle. The
        caller is responsible for stopping / waiting for the
        Proc instance.
        """
        if cmd is None:
            cmd = self.beat_path + "/packetbeat.test"

        args = [cmd,
                "-e",
                "-c", os.path.join(self.working_dir, config),
                "-systemTest",
                ]
        if os.getenv("TEST_COVERAGE") == "true":
            args += [
                "-test.coverprofile", os.path.join(self.working_dir, "coverage.cov"),
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

        self.expected_fields, self.dict_fields, _ = self.load_fields()
        super(BaseTest, self).setUp()
