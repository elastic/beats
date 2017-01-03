from filebeat import BaseTest
from beat.beat import INTEGRATION_TESTS
import os
import unittest
import glob
import subprocess
from elasticsearch import Elasticsearch
import json
import logging


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
                                        "/../../../../filebeat.py")

    # @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @unittest.skip("modules disabled in 5.2")
    def test_modules(self):
        self.init()
        modules = os.getenv("TESTING_FILEBEAT_MODULES")
        if modules:
            modules = modules.split(",")
        else:
            modules = os.listdir(self.modules_path)

        for module in modules:
            path = os.path.join(self.modules_path, module)
            filesets = [name for name in os.listdir(path) if
                        os.path.isfile(os.path.join(path, name,
                                                    "manifest.yml"))]

            for fileset in filesets:
                test_files = glob.glob(os.path.join(self.modules_path, module,
                                                    fileset, "test", "*.log"))
                for test_file in test_files:
                    self.run_on_file(
                        module=module,
                        fileset=fileset,
                        test_file=test_file)

    def run_on_file(self, module, fileset, test_file):
        print("Testing {}/{} on {}".format(module, fileset, test_file))

        index_name = "test-filebeat-modules"
        try:
            self.es.indices.delete(index=index_name)
        except:
            pass

        cmd = [
            self.filebeat,
            "--once",
            "--modules={}".format(module),
            "-M", "{module}.{fileset}.paths={test_file}".format(
                module=module, fileset=fileset, test_file=test_file),
            "--es", self.elasticsearch_url,
            "--index", index_name,
            "--registry", self.working_dir + "/registry"
        ]
        output = open(os.path.join(self.working_dir, "output.log"), "ab")
        subprocess.Popen(cmd,
                         stdin=None,
                         stdout=output,
                         stderr=subprocess.STDOUT,
                         bufsize=0).wait()

        # Make sure index exists
        self.wait_until(lambda: self.es.indices.exists(index_name))

        self.es.indices.refresh(index=index_name)
        res = self.es.search(index=index_name,
                             body={"query": {"match_all": {}}})
        objects = [o["_source"] for o in res["hits"]["hits"]]
        assert len(objects) > 0
        for obj in objects:
            self.assert_fields_are_documented(obj)

        if os.path.exists(test_file + "-expected.json"):
            with open(test_file + "-expected.json", "r") as f:
                expected = json.load(f)
                assert len(expected) == len(objects)
                for ev in expected:
                    found = False
                    for obj in objects:
                        if ev["_source"][module] == obj[module]:
                            found = True
                            break
                    if not found:
                        raise Exception("The following expected object was" +
                                        " not found: {}".format(obj))
