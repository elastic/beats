import os
import metricbeat
import unittest
from nose.plugins.skip import SkipTest
import urllib2
import json
import semver
import time


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['logstash']
    FIELDS = ['logstash']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_node(self):
        """
        logstash node metricset test
        """
        self.check_metricset("logstash", "node", self.get_hosts(), self.FIELDS + ["process"])

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_node_stats(self):
        """
        logstash node_stats metricset test
        """
        self.check_metricset("logstash", "node_stats", self.get_hosts(), self.FIELDS)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @unittest.skip("flaky; see https://github.com/elastic/beats/issues/13947")
    def test_xpack(self):
        """
        logstash-xpack module tests
        """
        version = self.get_version()
        if semver.compare(version, "7.3.0") == -1:
            # Skip for Logstash versions < 7.3.0 as necessary APIs not available
            raise SkipTest

        self.render_config_template(modules=[{
            "name": "logstash",
            "metricsets": ["node", "node_stats"],
            "hosts": self.get_hosts(),
            "period": "1s",
            "extras": {
                "xpack.enabled": "true"
            }
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

    def get_version(self):
        host = self.get_hosts()[0]
        res = urllib2.urlopen("http://" + host + "/").read()

        body = json.loads(res)
        version = body["version"]

        return version
