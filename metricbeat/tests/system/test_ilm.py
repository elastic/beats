from metricbeat import BaseTest
import os
from elasticsearch import Elasticsearch, TransportError
from nose.plugins.attrib import attr
import unittest

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class Test(BaseTest):

    #COMPOSE_SERVICES = ['elasticsearch']

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_ilm_enabled(self):
        """
        Test ilm sends data to correct index
        """

        self.render_config_template(
            modules=[{
                "name": "system",
                "metricsets": ["cpu"],
                "period": "1s"
            }],
            elasticsearch={
                "host": self.get_elasticsearch_url(),
                "ilm": True,
            },
        )

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("metricbeat start running."))
        self.wait_until(lambda: self.log_contains("Overwriting setup.template for ILM"))
        self.wait_until(lambda: self.log_contains("PublishEvents: 1 events have been published"))
        proc.check_kill_and_wait()

        es = Elasticsearch([self.get_elasticsearch_url()])

        # TODO: Get Version of metricbeat to fetch right indices / aliases

        # Check if template is loaded with settings
        template = es.transport.perform_request('GET', '/_template/metricbeat-7.0.0-alpha1')
        # TODO: check content of template if alias inside

        # Make sure the correct index + alias was created
        alias = es.transport.perform_request('GET', '/_alias/metricbeat-7.0.0-alpha1')
        assert "metricbeat-7.0.0-alpha1-000001" in alias

        # Asserts that data is actually written to the ILM indices
        data = es.transport.perform_request('GET', '/metricbeat-7.0.0-alpha1-000001/_search')
        assert data["hits"]["total"] > 0
