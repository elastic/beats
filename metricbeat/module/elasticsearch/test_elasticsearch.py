import re
import sys
import os
import unittest
from elasticsearch import Elasticsearch, TransportError
from parameterized import parameterized

sys.path.append(os.path.join(os.path.dirname(__file__), '../../tests/system'))

import metricbeat


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['elasticsearch']
    FIELDS = ["elasticsearch"]

    @parameterized.expand([
        "index",
        "index_summary",
        "node_stats",
        "node",
        "shard"
    ])
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_metricsets(self, metricset):
        """
        elasticsearch metricset tests
        """
        es = Elasticsearch(self.get_hosts())
        es.indices.create(index='test-index', ignore=400)
        self.check_metricset("elasticsearch", metricset, self.get_hosts(), self.FIELDS + ["service.name"])

    def get_hosts(self):
        return [os.getenv('ES_HOST', 'localhost') + ':' +
                os.getenv('ES_PORT', '9200')]
