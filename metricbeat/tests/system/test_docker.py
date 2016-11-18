import metricbeat

import unittest
from nose.plugins.attrib import attr

class Test(metricbeat.BaseTest):

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_container_fields(self):
        """
        test container fields
        """
        self.render_config_template(modules=[{
            "name": "docker",
            "metricsets": ["container"],
            "hosts": ["localhost"],
            "period": "1s",
            "socket": "unix:///var/run/docker.sock",
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log.replace("WARN EXPERIMENTAL", ""), "ERR|WARN")

        output = self.read_output_json()
        evt = output[0]

        evt = self.remove_labels_and_ports(evt)
        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_cpu_fields(self):
        """
        test cpu fields
        """
        self.render_config_template(modules=[{
            "name": "docker",
            "metricsets": ["cpu"],
            "hosts": ["localhost"],
            "period": "1s"
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=30)
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log.replace("WARN EXPERIMENTAL", ""), "ERR|WARN")

        output = self.read_output_json()
        evt = output[0]

        evt = self.remove_labels_and_ports(evt)

        if 'per_cpu' in evt["docker"]["cpu"]["usage"]:
            del evt["docker"]["cpu"]["usage"]["per_cpu"]

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_diskio_fields(self):
        """
        test diskio fields
        """
        self.render_config_template(modules=[{
            "name": "docker",
            "metricsets": ["diskio"],
            "hosts": ["localhost"],
            "period": "1s"
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=30)
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log.replace("WARN EXPERIMENTAL", ""), "ERR|WARN")

        output = self.read_output_json()
        evt = output[0]

        evt = self.remove_labels_and_ports(evt)

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_info_fields(self):
        """
        test info fields
        """
        self.render_config_template(modules=[{
            "name": "docker",
            "metricsets": ["info"],
            "hosts": ["localhost"],
            "period": "1s"
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=30)
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log.replace("WARN EXPERIMENTAL", ""), "ERR|WARN")

        output = self.read_output_json()
        evt = output[0]

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_memory_fields(self):
        """
        test memory fields
        """
        self.render_config_template(modules=[{
            "name": "docker",
            "metricsets": ["memory"],
            "hosts": ["localhost"],
            "period": "1s"
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=30)
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log.replace("WARN EXPERIMENTAL", ""), "ERR|WARN")

        output = self.read_output_json()
        evt = output[0]

        evt = self.remove_labels_and_ports(evt)
        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_network_fields(self):
        """
        test info fields
        """
        self.render_config_template(modules=[{
            "name": "docker",
            "metricsets": ["network"],
            "hosts": ["localhost"],
            "period": "1s"
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=30)
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log.replace("WARN EXPERIMENTAL", ""), "ERR|WARN")

        output = self.read_output_json()
        evt = output[0]

        evt = self.remove_labels_and_ports(evt)
        self.assert_fields_are_documented(evt)

    def remove_labels_and_ports(self, evt):

        if 'labels' in evt["docker"]["container"]:
            del evt["docker"]["container"]["labels"]
        if 'ports' in evt["docker"]["container"]:
            del evt["docker"]["container"]["ports"]

        return evt
