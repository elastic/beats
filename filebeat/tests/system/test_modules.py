from filebeat import BaseTest
from beat.beat import INTEGRATION_TESTS
import os
import unittest
import glob
import shutil
import subprocess
from elasticsearch import Elasticsearch
import json
import logging


class Test(BaseTest):

    def init(self):
        self.elasticsearch_url = self.get_elasticsearch_url()
        self.kibana_url = self.get_kibana_url()
        print("Using elasticsearch: {}".format(self.elasticsearch_url))
        self.es = Elasticsearch([self.elasticsearch_url])
        logging.getLogger("urllib3").setLevel(logging.WARNING)
        logging.getLogger("elasticsearch").setLevel(logging.ERROR)

        self.modules_path = os.path.abspath(self.working_dir +
                                            "/../../../../module")

        self.kibana_path = os.path.abspath(self.working_dir +
                                           "/../../../../_meta/kibana")

        self.filebeat = os.path.abspath(self.working_dir +
                                        "/../../../../filebeat.test")

        self.index_name = "test-filebeat-modules"

    @unittest.skipIf(not INTEGRATION_TESTS or
                     os.getenv("TESTING_ENVIRONMENT") == "2x",
                     "integration test not available on 2.x")
    def test_modules(self):
        """
        Tests all filebeat modules
        """
        self.init()
        modules = os.getenv("TESTING_FILEBEAT_MODULES")
        if modules:
            modules = modules.split(",")
        else:
            modules = os.listdir(self.modules_path)

        # generate a minimal configuration
        cfgfile = os.path.join(self.working_dir, "filebeat.yml")
        self.render_config_template(
            template_name="filebeat_modules",
            output=cfgfile,
            index_name=self.index_name,
            elasticsearch_url=self.elasticsearch_url
        )

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
                        test_file=test_file,
                        cfgfile=cfgfile)

    def _test_expected_events(self, module, test_file, res, objects):
        with open(test_file + "-expected.json", "r") as f:
            expected = json.load(f)

        if len(expected) > len(objects):
            res = self.es.search(index=self.index_name,
                                 body={"query": {"match_all": {}},
                                       "size": len(expected)})
            objects = [o["_source"] for o in res["hits"]["hits"]]

        assert len(expected) == res['hits']['total'], "expected {} but got {}".format(
            len(expected), res['hits']['total'])

        for ev in expected:
            found = False
            for obj in objects:
                if ev["_source"][module] == obj[module]:
                    found = True
                    break

            assert found, "The following expected object was not found:\n {}\nSearched in: \n{}".format(
                ev["_source"][module], objects)

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
            "-modules={}".format(module),
            "-M", "{module}.*.enabled=false".format(module=module),
            "-M", "{module}.{fileset}.enabled=true".format(module=module, fileset=fileset),
            "-M", "{module}.{fileset}.var.paths=[{test_file}]".format(
                module=module, fileset=fileset, test_file=test_file),
            "-M", "*.*.input.close_eof=true",
        ]

        output_path = os.path.join(self.working_dir, module, fileset, os.path.basename(test_file))
        if not os.path.exists(output_path):
            os.makedirs(output_path)

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
        res = self.es.search(index=self.index_name,
                             body={"query": {"match_all": {}}})
        objects = [o["_source"] for o in res["hits"]["hits"]]
        assert len(objects) > 0
        for obj in objects:
            assert obj["fileset"]["module"] == module, "expected fileset.module={} but got {}".format(
                module, obj["fileset"]["module"])

            assert "error" not in obj, "not error expected but got: {}".format(obj)

            if (module == "auditd" and fileset == "log") \
                    or (module == "osquery" and fileset == "result"):
                # There are dynamic fields that are not documented.
                pass
            else:
                self.assert_fields_are_documented(obj)

        if os.path.exists(test_file + "-expected.json"):
            self._test_expected_events(module, test_file, res, objects)

    @unittest.skipIf(not INTEGRATION_TESTS or
                     os.getenv("TESTING_ENVIRONMENT") == "2x",
                     "integration test not available on 2.x")
    def test_input_pipeline_config(self):
        """
        Tests that the pipeline configured in the input overwrites
        the one from the output.
        """
        self.init()
        index_name = "filebeat-test-input"
        try:
            self.es.indices.delete(index=index_name)
        except:
            pass
        self.wait_until(lambda: not self.es.indices.exists(index_name))

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            elasticsearch=dict(
                host=self.elasticsearch_url,
                pipeline="estest",
                index=index_name),
            pipeline="test",
            setup_template_name=index_name,
            setup_template_pattern=index_name + "*",
        )

        os.mkdir(self.working_dir + "/log/")
        testfile = self.working_dir + "/log/test.log"
        with open(testfile, 'a') as file:
            file.write("Hello World1\n")

        # put pipeline
        self.es.transport.perform_request("PUT", "/_ingest/pipeline/test",
                                          body={
                                              "processors": [{
                                                  "set": {
                                                      "field": "x-pipeline",
                                                      "value": "test-pipeline",
                                                  }
                                              }]})

        filebeat = self.start_beat()

        # Wait until the event is in ES
        self.wait_until(lambda: self.es.indices.exists(index_name))

        def search_objects():
            try:
                self.es.indices.refresh(index=index_name)
                res = self.es.search(index=index_name,
                                     body={"query": {"match_all": {}}})
                return [o["_source"] for o in res["hits"]["hits"]]
            except:
                return []

        self.wait_until(lambda: len(search_objects()) > 0, max_timeout=20)
        filebeat.check_kill_and_wait()

        objects = search_objects()
        assert len(objects) == 1
        o = objects[0]
        assert o["x-pipeline"] == "test-pipeline"

    @unittest.skipIf(not INTEGRATION_TESTS or
                     os.getenv("TESTING_ENVIRONMENT") == "2x",
                     "integration test not available on 2.x")
    def test_ml_setup(self):
        """ Test ML are installed in all possible ways """
        for setup_flag in (True, False):
            for modules_flag in (True, False):
                self._run_ml_test(setup_flag, modules_flag)

    def _run_ml_test(self, setup_flag, modules_flag):
        self.init()

        # Clean any previous state
        for df in self.es.transport.perform_request("GET", "/_xpack/ml/datafeeds/")["datafeeds"]:
            if df["datafeed_id"] == 'filebeat-nginx-access-response_code':
                self.es.transport.perform_request("DELETE", "/_xpack/ml/datafeeds/" + df["datafeed_id"])

        for df in self.es.transport.perform_request("GET", "/_xpack/ml/anomaly_detectors/")["jobs"]:
            if df["job_id"] == 'datafeed-filebeat-nginx-access-response_code':
                self.es.transport.perform_request("DELETE", "/_xpack/ml/anomaly_detectors/" + df["job_id"])

        shutil.rmtree(os.path.join(self.working_dir, "modules.d"), ignore_errors=True)

        # generate a minimal configuration
        cfgfile = os.path.join(self.working_dir, "filebeat.yml")
        self.render_config_template(
            template_name="filebeat_modules",
            output=cfgfile,
            index_name=self.index_name,
            elasticsearch_url=self.elasticsearch_url,
            kibana_url=self.kibana_url,
            kibana_path=self.kibana_path)

        if not modules_flag:
            # Enable nginx
            os.mkdir(os.path.join(self.working_dir, "modules.d"))
            with open(os.path.join(self.working_dir, "modules.d/nginx.yml"), "wb") as nginx:
                nginx.write("- module: nginx")

        cmd = [
            self.filebeat, "-systemTest",
            "-e", "-d", "*",
            "-c", cfgfile
        ]

        if setup_flag:
            cmd += ["--setup"]
        else:
            cmd += ["setup", "--machine-learning"]

        if modules_flag:
            cmd += ["--modules=nginx"]

        output_path = os.path.join(self.working_dir, "output.log")
        output = open(output_path, "ab")
        output.write(" ".join(cmd) + "\n")
        beat = subprocess.Popen(cmd,
                                stdin=None,
                                stdout=output,
                                stderr=output,
                                bufsize=0)

        # Check result
        self.wait_until(lambda: "filebeat-nginx-access-response_code" in
                        (df["job_id"] for df in self.es.transport.perform_request(
                            "GET", "/_xpack/ml/anomaly_detectors/")["jobs"]),
                        max_timeout=30)
        self.wait_until(lambda: "datafeed-filebeat-nginx-access-response_code" in
                        (df["datafeed_id"] for df in self.es.transport.perform_request("GET", "/_xpack/ml/datafeeds/")["datafeeds"]))

        beat.kill()

        # check if fails during trying to setting it up again
        output = open(output_path, "ab")
        output.write(" ".join(cmd) + "\n")
        beat = subprocess.Popen(cmd,
                                stdin=None,
                                stdout=output,
                                stderr=output,
                                bufsize=0)

        output = open(output_path, "r")
        for obj in ["Datafeed", "Job", "Dashboard", "Search", "Visualization"]:
            self.wait_log_contains("{obj} already exists".format(obj=obj),
                                   logfile=output_path,
                                   max_timeout=30)

        beat.kill()
