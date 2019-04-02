from filebeat import BaseTest
from beat.beat import INTEGRATION_TESTS
import os
import unittest
import glob
import subprocess

from elasticsearch import Elasticsearch
import json
import logging
from parameterized import parameterized


def load_fileset_test_cases():
    """
    Creates a list of all modules, filesets and testfiles inside for testing.
    To execute tests for only 1 module, set the env variable TESTING_FILEBEAT_MODULES
    to the specific module name or a , separated lists of modules.
    """
    modules_dir = os.getenv("MODULES_PATH")
    if not modules_dir:
        current_dir = os.path.dirname(os.path.abspath(__file__))
        modules_dir = os.path.join(current_dir, "..", "..", "module")
    modules = os.getenv("TESTING_FILEBEAT_MODULES")
    if modules:
        modules = modules.split(",")
    else:
        modules = os.listdir(modules_dir)

    filesets_env = os.getenv("TESTING_FILEBEAT_FILESETS")

    test_cases = []

    for module in modules:
        path = os.path.join(modules_dir, module)

        if not os.path.isdir(path):
            continue

        if filesets_env:
            filesets = filesets_env.split(",")
        else:
            filesets = os.listdir(path)

        for fileset in filesets:
            if not os.path.isdir(os.path.join(path, fileset)):
                continue

            if not os.path.isfile(os.path.join(path, fileset, "manifest.yml")):
                continue

            test_files = glob.glob(os.path.join(modules_dir, module,
                                                fileset, "test", "*.log"))
            for test_file in test_files:
                test_cases.append([module, fileset, test_file])

    return test_cases


class Test(BaseTest):

    def init(self):
        self.elasticsearch_url = self.get_elasticsearch_url()
        print("Using elasticsearch: {}".format(self.elasticsearch_url))
        self.es = Elasticsearch([self.elasticsearch_url])
        logging.getLogger("urllib3").setLevel(logging.WARNING)
        logging.getLogger("elasticsearch").setLevel(logging.ERROR)

        self.modules_path = os.path.abspath(self.working_dir +
                                            "/../../../../module")

        self.filebeat = os.path.abspath(self.working_dir +
                                        "/../../../../filebeat.test")

        self.index_name = "test-filebeat-modules"

        body = {
            "transient": {
                "script.max_compilations_rate": "1000/1m"
            }
        }

        self.es.transport.perform_request('PUT', "/_cluster/settings", body=body)

    @parameterized.expand(load_fileset_test_cases)
    @unittest.skipIf(not INTEGRATION_TESTS,
                     "integration tests are disabled, run with INTEGRATION_TESTS=1 to enable them.")
    @unittest.skipIf(os.getenv("TESTING_ENVIRONMENT") == "2x",
                     "integration test not available on 2.x")
    def test_fileset_file(self, module, fileset, test_file):
        self.init()

        # generate a minimal configuration
        cfgfile = os.path.join(self.working_dir, "filebeat.yml")
        self.render_config_template(
            template_name="filebeat_modules",
            output=cfgfile,
            index_name=self.index_name,
            elasticsearch_url=self.elasticsearch_url,
        )

        self.run_on_file(
            module=module,
            fileset=fileset,
            test_file=test_file,
            cfgfile=cfgfile)

    def run_on_file(self, module, fileset, test_file, cfgfile):
        print("Testing {}/{} on {}".format(module, fileset, test_file))

        try:
            self.es.indices.delete(index=self.index_name)
        except:
            pass
        self.wait_until(lambda: not self.es.indices.exists(self.index_name))

        cmd = [
            self.filebeat, "-systemTest",
            "-e", "-d", "*", "-once",
            "-c", cfgfile,
            "-E", "setup.ilm.enabled=false",
            "-modules={}".format(module),
            "-M", "{module}.*.enabled=false".format(module=module),
            "-M", "{module}.{fileset}.enabled=true".format(
                module=module, fileset=fileset),
            "-M", "{module}.{fileset}.var.input=file".format(
                module=module, fileset=fileset),
            "-M", "{module}.{fileset}.var.paths=[{test_file}]".format(
                module=module, fileset=fileset, test_file=test_file),
            "-M", "*.*.input.close_eof=true",
        ]

        # Based on the convention that if a name contains -json the json format is needed. Currently used for LS.
        if "-json" in test_file:
            cmd.append("-M")
            cmd.append("{module}.{fileset}.var.format=json".format(module=module, fileset=fileset))

        output_path = os.path.join(self.working_dir)
        output = open(os.path.join(output_path, "output.log"), "ab")
        output.write(" ".join(cmd) + "\n")
        subprocess.Popen(cmd,
                         stdin=None,
                         stdout=output,
                         stderr=subprocess.STDOUT,
                         bufsize=0).wait()

        # Make sure index exists
        self.wait_until(lambda: self.es.indices.exists(self.index_name))

        self.es.indices.refresh(index=self.index_name)
        # Loads the first 100 events to be checked
        res = self.es.search(index=self.index_name,
                             body={"query": {"match_all": {}}, "size": 100, "sort": {"log.offset": {"order": "asc"}}})
        objects = [o["_source"] for o in res["hits"]["hits"]]
        assert len(objects) > 0
        for obj in objects:
            assert obj["event"]["module"] == module, "expected event.module={} but got {}".format(
                module, obj["event"]["module"])

            assert "error" not in obj, "not error expected but got: {}".format(
                obj)

            if (module == "auditd" and fileset == "log") \
                    or (module == "osquery" and fileset == "result"):
                # There are dynamic fields that are not documented.
                pass
            else:
                self.assert_fields_are_documented(obj)

        if os.path.exists(test_file + "-expected.json"):
            self._test_expected_events(test_file, objects)

    def _test_expected_events(self, test_file, objects):

        # Generate expected files if GENERATE env variable is set
        if os.getenv("GENERATE"):
            with open(test_file + "-expected.json", 'w') as f:
                # Flatten an cleanup objects
                # This makes sure when generated on different machines / version the expected.json stays the same.
                for k, obj in enumerate(objects):
                    objects[k] = self.flatten_object(obj, {}, "")
                    clean_keys(objects[k])

                json.dump(objects, f, indent=4, separators=(',', ': '), sort_keys=True)

        with open(test_file + "-expected.json", "r") as f:
            expected = json.load(f)

        assert len(expected) == len(objects), "expected {} events to compare but got {}".format(
            len(expected), len(objects))

        for ev in expected:
            found = False
            for obj in objects:

                # Flatten objects for easier comparing
                obj = self.flatten_object(obj, {}, "")
                clean_keys(obj)

                if ev == obj:
                    found = True
                    break

            assert found, "The following expected object was not found:\n {}\nSearched in: \n{}".format(
                pretty_json(ev), pretty_json(objects))


def clean_keys(obj):
    # These keys are host dependent
    host_keys = ["host.name", "agent.hostname", "agent.type", "agent.ephemeral_id", "agent.id"]
    # The create timestamps area always new
    time_keys = ["event.created"]
    # source path and agent.version can be different for each run
    other_keys = ["log.file.path", "agent.version"]

    for key in host_keys + time_keys + other_keys:
        delete_key(obj, key)

    # Remove timestamp for comparison where timestamp is not part of the log line
    if (obj["event.dataset"] in ["icinga.startup", "redis.log", "haproxy.log", "system.auth", "system.syslog"]):
        delete_key(obj, "@timestamp")


def delete_key(obj, key):
    if key in obj:
        del obj[key]


def pretty_json(obj):
    return json.dumps(obj, indent=2, separators=(',', ': '))
