import os
import metricbeat
import unittest


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['ceph']
    COMPOSE_TIMEOUT = 300

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_cluster_disk(self):
        """
        ceph cluster_disk metricset test
        """
        self.render_config_template(modules=[{
            "name": "ceph",
            "metricsets": ["cluster_disk"],
            "hosts": self.get_hosts(),
            "period": "1s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()

        output = self.read_output_json()
        self.assertTrue(len(output) >= 1)
        evt = output[0]
        print evt

        self.assertTrue("error" not in evt)
        self.assertTrue("ceph" in evt)
        self.assertTrue("cluster_disk" in evt["ceph"])

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_cluster_health(self):
        """
        ceph cluster_health metricset test
        """
        self.render_config_template(modules=[{
            "name": "ceph",
            "metricsets": ["cluster_health"],
            "hosts": self.get_hosts(),
            "period": "1s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()

        output = self.read_output_json()
        self.assertTrue(len(output) >= 1)
        evt = output[0]
        print evt

        self.assertTrue("error" not in evt)
        self.assertTrue("ceph" in evt)
        self.assertTrue("cluster_health" in evt["ceph"])

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_monitor_health(self):
        """
        ceph monitor_health metricset test
        """
        self.render_config_template(modules=[{
            "name": "ceph",
            "metricsets": ["monitor_health"],
            "hosts": self.get_hosts(),
            "period": "1s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()

        output = self.read_output_json()
        self.assertTrue(len(output) >= 1)
        evt = output[0]
        print evt

        self.assertTrue("error" not in evt)
        self.assertTrue("ceph" in evt)
        self.assertTrue("monitor_health" in evt["ceph"])

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_pool_disk(self):
        """
        ceph pool_disk metricset test
        """
        self.render_config_template(modules=[{
            "name": "ceph",
            "metricsets": ["pool_disk"],
            "hosts": self.get_hosts(),
            "period": "1s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()

        output = self.read_output_json()
        self.assertTrue(len(output) >= 1)
        evt = output[0]
        print evt

        self.assertTrue("error" not in evt)
        self.assertTrue("ceph" in evt)
        self.assertTrue("pool_disk" in evt["ceph"])

        self.assert_fields_are_documented(evt)

    def get_hosts(self):
        return [os.getenv('CEPH_HOST', 'localhost') + ':' +
                os.getenv('CEPH_PORT', '5000')]
