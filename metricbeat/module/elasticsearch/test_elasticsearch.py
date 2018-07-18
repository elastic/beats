import re
import sys
import os
import unittest
from elasticsearch import Elasticsearch, TransportError
from parameterized import parameterized
import json

sys.path.append(os.path.join(os.path.dirname(__file__), '../../tests/system'))

import metricbeat


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['elasticsearch']
    FIELDS = ["elasticsearch"]

    @parameterized.expand([
        "index",
        "index_summary",
        "ml_job",
        "index_recovery",
        "node_stats",
        "node",
        "shard"
    ])
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_metricsets(self, metricset):
        """
        elasticsearch metricset tests
        """
        if metricset == "ml_job":
            self.create_ml_job()

        es = Elasticsearch(self.get_hosts())
        es.indices.create(index='test-index', ignore=400)
        self.check_metricset("elasticsearch", metricset, self.get_hosts(), self.FIELDS +
                             ["service.name"], extras={"index_recovery.active_only": "false"})

    def get_hosts(self):
        return [os.getenv('ES_HOST', 'localhost') + ':' +
                os.getenv('ES_PORT', '9200')]

    def create_ml_job(self):
        es = Elasticsearch(self.get_hosts())

        # Enable xpack trial
        try:
            es.transport.perform_request('POST', "/_xpack/license/start_trial?acknowledge=true")
        except:
            e = sys.exc_info()[0]
            print "Trial already enabled. Error: {}".format(e)

        # Check if an ml job already exists
        response = es.transport.perform_request('GET', "/_xpack/ml/anomaly_detectors/_all/")
        if response["count"] > 0:
            return

        file = os.path.join(self.beat_path, "module", "elasticsearch", "ml_job", "_meta", "test", "test_job.json")

        body = {}
        with open(file, 'r') as f:
            body = json.load(f)

        path = "/_xpack/ml/anomaly_detectors/test"
        es.transport.perform_request('PUT', path, body=body)
