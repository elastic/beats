from base import BaseTest
from elasticsearch import Elasticsearch, TransportError

import logging
import os


INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class TestCommandVersion(BaseTest):
    """
    Test beat subcommands
    """

    def setUp(self):
        super(BaseTest, self).setUp()

        self.elasticsearch_url = self.get_elasticsearch_url()
        print("Using elasticsearch: {}".format(self.elasticsearch_url))
        self.es = Elasticsearch([self.elasticsearch_url])
        logging.getLogger("urllib3").setLevel(logging.WARNING)
        logging.getLogger("elasticsearch").setLevel(logging.ERROR)

    def test_version(self):
        """
        Test version command
        """
        exit_code = self.run_beat(
            extra_args=["version"], logging_args=["-v", "-d", "*"])
        assert exit_code == 0

        assert self.log_contains("mockbeat")
        assert self.log_contains("version")
        assert self.log_contains("9.9.9")
