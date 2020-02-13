import os
import requests
import sys
import time
import unittest
from parameterized import parameterized

sys.path.append(os.path.join(os.path.dirname(__file__), '../../tests/system'))
import metricbeat


class Test(metricbeat.BaseTest):
    COMPOSE_SERVICES = ['ceph-mgr']
    FIELDS = ["ceph"]

    def get_ceph_module_config(self, metricset):
        return {
            'name': 'ceph',
            'metricsets': [metricset],
            'period': '1h',
            'hosts': self.get_hosts(),
            'username': 'demo',
            'password': self.get_password(),
            'extras': {
                'ssl.verification_mode': 'none'
            }
        }

    def get_hosts(self):
        return ['https://' + self.compose_host(port='8003/tcp')]

    def get_password(self):
        r = requests.get('http://' + self.compose_host(port='5000/tcp') + '/restful-list-keys.json')
        keys = r.json()
        return keys['demo']

    @parameterized.expand([
        "mgr_cluster_disk",
        "mgr_cluster_health",
        "mgr_osd_disk",
        "mgr_osd_perf",
        "mgr_osd_pool_stats",
        "mgr_osd_tree"
    ])
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_ceph_mgr(self, metricset):
        """
        ceph-mgr metricsets tests
        """

        self.render_config_template(modules=[self.get_ceph_module_config(metricset)])
        proc = self.start_beat(home=self.beat_path)
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings(replace=['SSL/TLS verifications disabled.'])

        output = self.read_output_json()
        for evt in output:
            assert 'ceph' in evt
            self.assert_fields_are_documented(evt)
