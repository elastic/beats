import metricbeat

import unittest
import os
from nose.plugins.attrib import attr


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['docker']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_container_fields(self):
        """
        test container fields
        """
        self.render_config_template(
            modules=[{
                "name": "docker",
                "metricsets": ["container"],
                "hosts": ["unix:///var/run/docker.sock"],
                "period": "10s",
            }],
        )

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings(["Container stopped when recovering stats",
                                        "An error occurred while getting docker stats"])

        output = self.read_output_json()
        evt = output[0]

        evt = self.remove_labels(evt)
        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_cpu_fields(self):
        """
        test cpu fields
        """
        self.render_config_template(modules=[{
            "name": "docker",
            "metricsets": ["cpu"],
            "hosts": ["unix:///var/run/docker.sock"],
            "period": "10s"
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=30)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings(["Container stopped when recovering stats",
                                        "An error occurred while getting docker stats"])

        output = self.read_output_json()
        evt = output[0]

        evt = self.remove_labels(evt)

        if 'core' in evt["docker"]["cpu"]:
            del evt["docker"]["cpu"]["core"]

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_diskio_fields(self):
        """
        test diskio fields
        """
        self.render_config_template(modules=[{
            "name": "docker",
            "metricsets": ["diskio"],
            "hosts": ["unix:///var/run/docker.sock"],
            "period": "10s"
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=30)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings(["Container stopped when recovering stats",
                                        "An error occurred while getting docker stats"])

        output = self.read_output_json()
        evt = output[0]

        evt = self.remove_labels(evt)

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_info_fields(self):
        """
        test info fields
        """
        self.render_config_template(modules=[{
            "name": "docker",
            "metricsets": ["info"],
            "hosts": ["unix:///var/run/docker.sock"],
            "period": "10s"
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=30)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings(["Container stopped when recovering stats",
                                        "An error occurred while getting docker stats"])

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
            "hosts": ["unix:///var/run/docker.sock"],
            "period": "10s"
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=30)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings(["Container stopped when recovering stats",
                                        "An error occurred while getting docker stats"])

        output = self.read_output_json()
        evt = output[0]

        evt = self.remove_labels(evt)
        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_network_fields(self):
        """
        test network fields
        """
        self.render_config_template(modules=[{
            "name": "docker",
            "metricsets": ["network"],
            "hosts": ["unix:///var/run/docker.sock"],
            "period": "10s"
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=30)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings(["Container stopped when recovering stats",
                                        "An error occurred while getting docker stats"])

        output = self.read_output_json()
        evt = output[0]

        evt = self.remove_labels(evt)
        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_health_fields(self):
        """
        test health fields
        """
        self.render_config_template(modules=[{
            "name": "docker",
            "metricsets": ["healthcheck"],
            "hosts": ["unix:///var/run/docker.sock"],
            "period": "10s",
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings(["Container stopped when recovering stats",
                                        "An error occurred while getting docker stats"])

        output = self.read_output_json()
        evt = output[0]

        evt = self.remove_labels(evt)
        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_image_fields(self):
        """
        test image fields
        """
        self.render_config_template(modules=[{
            "name": "docker",
            "metricsets": ["image"],
            "hosts": ["unix:///var/run/docker.sock"],
            "period": "10s",
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings(["Container stopped when recovering stats",
                                        "An error occurred while getting docker stats"])

        output = self.read_output_json()
        evt = output[0]

        if 'tags' in evt["docker"]["image"]:
            del evt["docker"]["image"]["tags"]

        if 'labels' in evt["docker"]["image"]:
            del evt["docker"]["image"]["labels"]

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_event_fields(self):
        """
        test event fields
        """
        self.render_config_template(modules=[{
            "name": "docker",
            "metricsets": ["event"],
            "hosts": ["unix:///var/run/docker.sock"],
            "period": "10s",
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings(["Container stopped when recovering stats",
                                        "An error occurred while getting docker stats"])

        output = self.read_output_json()
        evt = output[0]

        if 'attributes' in evt["docker"]["event"]["actor"]:
            del evt["docker"]["event"]["actor"]["attributes"]

        self.assert_fields_are_documented(evt)

    def remove_labels(self, evt):

        if 'labels' in evt["docker"]["container"]:
            del evt["docker"]["container"]["labels"]

        return evt
