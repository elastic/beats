from base import BaseTest
import os
from elasticsearch import Elasticsearch, TransportError
from nose.plugins.attrib import attr
import unittest

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class Test(BaseTest):

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_ilm_enabled(self):
        """
        Test ilm enabled
        """

        self.render_config_template(
            elasticsearch={
                "hosts": self.get_host(),
                "ilm.enabled": True,
            },
        )

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("Overwriting setup.template for ILM"))
        proc.check_kill_and_wait()

        es = Elasticsearch([self.get_elasticsearch_url()])

        # Check if template is loaded with settings
        template = es.transport.perform_request('GET', '/_template/mockbeat-9.9.9')
        print template

        # Make sure the correct index + alias was created
        alias = es.transport.perform_request('GET', '/_alias/mockbeat-9.9.9')
        assert "mockbeat-9.9.9-0001" in alias

        # TODO: How do we check that data is sent to the alias -> implement in metricbeat to have data?

    def get_host(self):
        return os.getenv('ES_HOST', 'localhost') + ':' + os.getenv('ES_PORT', '9200')
