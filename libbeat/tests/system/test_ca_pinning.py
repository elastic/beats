import logging
import os
import pytest
import unittest
from base import BaseTest
from elasticsearch import RequestError
from idxmgmt import IdxMgmt

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class TestCAPinning(BaseTest):
    """
    Test beat CA pinning for elasticsearch
    """

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_sending_events_with_a_good_sha256(self):
        """
        Test Sending events while using ca pinning with a good sha256
        """

        ca = os.path.join(self.beat_path,
                          "..",
                          "testing",
                          "environments",
                          "docker",
                          "elasticsearch",
                          "pki",
                          "ca",
                          "ca.crt")

        self.render_config_template(
            elasticsearch={
                "host": self.get_elasticsearch_url_ssl(),
                "user": "admin",
                "pass": "testing",
                "ssl_certificate_authorities": [ca],
                "ssl_ca_sha256": "8hZS8gpciuzlu+7Xi0sdv8T7RKRRxG1TWKumUQsDam0=",
            },
        )

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("PublishEvents: 1 events have been published"))
        proc.check_kill_and_wait()

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_sending_events_with_a_bad_sha256(self):
        """
        Test Sending events while using ca pinning with a bad sha256
        """

        ca = os.path.join(self.beat_path,
                          "..",
                          "testing",
                          "environments",
                          "docker",
                          "elasticsearch",
                          "pki",
                          "ca",
                          "ca.crt")

        self.render_config_template(
            elasticsearch={
                "host": self.get_elasticsearch_url_ssl(),
                "user": "beats",
                "pass": "testing",
                "ssl_certificate_authorities": [ca],
                "ssl_ca_sha256": "not-good-sha",
            },
        )

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains(
            "provided CA certificate pins doesn't match any of the certificate authorities used to validate the certificate"))
        proc.check_kill_and_wait()
