from filebeat import BaseTest
from beat.beat import INTEGRATION_TESTS
import os
import unittest
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
                                           "/../../../../_meta/kibana.generated")

        self.filebeat = os.path.abspath(self.working_dir +
                                        "/../../../../filebeat.test")

        self.index_name = "test-filebeat-pipeline"

    @unittest.skipIf(not INTEGRATION_TESTS,
                     "integration tests are disabled, run with INTEGRATION_TESTS=1 to enable them.")
    @unittest.skipIf(os.getenv("TESTING_ENVIRONMENT") == "2x",
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

        body = {
            "transient": {
                "script.max_compilations_rate": "100/1m"
            }
        }

        self.es.transport.perform_request('PUT', "/_cluster/settings", body=body)

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            elasticsearch=dict(
                host=self.elasticsearch_url,
                pipeline="estest",
                index=index_name),
            pipeline="test",
            setup_template_name=index_name,
            setup_template_pattern=index_name + "*",
            ilm={"enabled": False},
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
