from filebeat import BaseTest
from beat.beat import INTEGRATION_TESTS
import os
import unittest
import shutil
import subprocess
from elasticsearch import Elasticsearch
import logging
from parameterized import parameterized
import semver


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
                                           "/../../../../build/kibana")

        self.filebeat = os.path.abspath(self.working_dir +
                                        "/../../../../filebeat.test")

        self.index_name = "test-filebeat-ml"

    @parameterized.expand([
        (False,),
        (True,),
    ])
    @unittest.skipIf(not INTEGRATION_TESTS,
                     "integration tests are disabled, run with INTEGRATION_TESTS=1 to enable them.")
    @unittest.skipIf(os.getenv("TESTING_ENVIRONMENT") == "2x",
                     "integration test not available on 2.x")
    @unittest.skipIf(os.name == "nt", "skipped on Windows")
    @unittest.skip("Skipped as flaky: https://github.com/elastic/beats/issues/11629")
    def test_ml_setup(self, modules_flag):
        """ Test ML are installed in all possible ways """
        self._run_ml_test(modules_flag)

    def _run_ml_test(self, modules_flag):
        self.init()

        from elasticsearch import AuthorizationException

        es_info = self.es.info()
        version = semver.parse(es_info["version"]["number"])
        if version["major"] < 7:
            start_trial_api_url = "/_xpack/license/start_trial?acknowledge=true"
            ml_datafeeds_url = "/_xpack/ml/datafeeds/"
            ml_anomaly_detectors_url = "/_xpack/ml/anomaly_detectors/"
        else:
            start_trial_api_url = "/_license/start_trial?acknowledge=true"
            ml_datafeeds_url = "/_ml/datafeeds/"
            ml_anomaly_detectors_url = "/_ml/anomaly_detectors/"

        try:
            output = self.es.transport.perform_request("POST", start_trial_api_url)
        except AuthorizationException:
            print("License already enabled")

        print("Test modules_flag: {}".format(modules_flag))

        # Clean any previous state
        for df in self.es.transport.perform_request("GET", ml_datafeeds_url)["datafeeds"]:
            if df["datafeed_id"] == 'filebeat-nginx-access-response_code':
                self.es.transport.perform_request(
                    "DELETE", "/_ml/datafeeds/" + df["datafeed_id"])

        for df in self.es.transport.perform_request("GET", ml_anomaly_detectors_url)["jobs"]:
            if df["job_id"] == 'datafeed-filebeat-nginx-access-response_code':
                self.es.transport.perform_request(
                    "DELETE", ml_anomaly_detectors_url + df["job_id"])

        shutil.rmtree(os.path.join(self.working_dir,
                                   "modules.d"), ignore_errors=True)

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

        # Skipping dashboard loading to speed up tests
        cmd += ["-E", "setup.dashboards.enabled=false"]
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
        self.wait_until(lambda: "filebeat-nginx_ecs-access-status_code_rate_ecs" in
                                (df["job_id"] for df in self.es.transport.perform_request(
                                    "GET", ml_anomaly_detectors_url)["jobs"]),
                        max_timeout=60)
        self.wait_until(lambda: "datafeed-filebeat-nginx_ecs-access-status_code_rate_ecs" in
                                (df["datafeed_id"] for df in self.es.transport.perform_request("GET", ml_datafeeds_url)["datafeeds"]))

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
                                   max_timeout=60)

        beat.kill()
