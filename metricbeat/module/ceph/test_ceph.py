import metricbeat
import os
import requests
import sys
import time
import unittest
from parameterized import parameterized


@metricbeat.parameterized_with_supported_versions
@unittest.skip("broken test suite: https://github.com/elastic/beats/issues/32898")
class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['ceph']
    FIELDS = ["ceph"]

    @parameterized.expand([
        "cluster_disk",
        "cluster_health",
        "monitor_health",
        "pool_disk",
    ])
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_ceph(self, metricset):
        """
        ceph metricsets tests
        """

        if not self.old_ceph_version():
            self.skipTest("newer ceph version not supported")
            return
        self.check_metricset("ceph", metricset, ['http://' + self.compose_host(port='5000/tcp')], self.FIELDS)

    @parameterized.expand([
        "mgr_cluster_disk",
        "mgr_cluster_health",
        "mgr_osd_perf",
        "mgr_osd_pool_stats",
        "mgr_osd_tree",
        "mgr_pool_disk"
    ])
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_ceph_mgr(self, metricset):
        """
        ceph metricsets tests
        """

        if self.old_ceph_version():
            self.skipTest("legacy ceph version not supported")
            return

        self.render_config_template(modules=[self.get_ceph_mgr_module_config(metricset)])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings(replace=['SSL/TLS verifications disabled.'])

        output = self.read_output_json()
        for evt in output:
            assert 'ceph' in evt
            self.assert_fields_are_documented(evt)

    def old_ceph_version(self):
        if not 'CEPH_CODENAME' in self.COMPOSE_ENV:
            return False

        return self.COMPOSE_ENV['CEPH_CODENAME'] == 'jewel'

    def get_ceph_mgr_module_config(self, metricset):
        return {
            'name': 'ceph',
            'metricsets': [metricset],
            'period': '1h',
            'hosts': ['https://' + self.compose_host(port='8003/tcp')],
            'username': 'demo',
            'password': self.get_ceph_mgr_password(),
            'extras': {
                'ssl.verification_mode': 'none'
            }
        }

    def get_ceph_mgr_password(self):
        r = requests.get('http://' + self.compose_host(port='5000/tcp') + '/restful-list-keys.json')
        keys = r.json()
        return keys['demo']
