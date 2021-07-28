import metricbeat
import os
import sys
import unittest


VSPHERE_FIELDS = metricbeat.COMMON_FIELDS + ["vsphere"]


@metricbeat.parameterized_with_supported_versions
class TestVsphere(metricbeat.BaseTest):
    COMPOSE_SERVICES = ['vsphere']

    @classmethod
    def get_hosts(cls):
        return ['https://{}/sdk'.format(cls.compose_host())]

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_datastore(self):
        """
        vsphere datastore test
        """
        self.render_config_template(modules=[{
            "name": "vsphere",
            "metricsets": ["datastore"],
            "hosts": self.get_hosts(),
            "period": "5s",
            "username": "user",
            "password": "pass",
            "extras": {
                "insecure": True,
            },
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.assertCountEqual(self.de_dot(VSPHERE_FIELDS), evt.keys(), evt)

        self.assertEqual(evt["service"]["address"], self.get_hosts()[0])

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_host(self):
        """
        vsphere host test
        """
        self.render_config_template(modules=[{
            "name": "vsphere",
            "metricsets": ["host"],
            "hosts": self.get_hosts(),
            "period": "5s",
            "username": "user",
            "password": "pass",
            "extras": {
                "insecure": True,
            },
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 4)
        evt = output[0]

        self.assertCountEqual(self.de_dot(VSPHERE_FIELDS), evt.keys(), evt)

        self.assertEqual(evt["service"]["address"], self.get_hosts()[0])

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_virtualmachine(self):
        """
        vsphere virtualmachine test
        """
        self.render_config_template(modules=[{
            "name": "vsphere",
            "metricsets": ["virtualmachine"],
            "hosts": self.get_hosts(),
            "period": "5s",
            "username": "user",
            "password": "pass",
            "extras": {
                "insecure": True,
            },
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 4)
        evt = output[0]

        self.assertCountEqual(self.de_dot(VSPHERE_FIELDS), evt.keys(), evt)

        self.assertEqual(evt["service"]["address"], self.get_hosts()[0])

        self.assert_fields_are_documented(evt)
