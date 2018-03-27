import os
import metricbeat
import unittest

from time import sleep
from beat.beat import INTEGRATION_TESTS


class TestAutodiscover(metricbeat.BaseTest):
    """
    Test metricbeat autodiscover
    """
    @unittest.skipIf(not INTEGRATION_TESTS or
                     os.getenv("TESTING_ENVIRONMENT") == "2x",
                     "integration test not available on 2.x")
    def test_docker(self):
        """
        Test docker autodiscover starts modules from templates
        """
        import docker
        docker_client = docker.from_env()

        self.render_config_template(
            autodiscover={
                'docker': {
                    'templates': '''
                      - condition:
                          equals.docker.container.image: memcached:1.5.3
                        config:
                          - module: memcached
                            metricsets: ["stats"]
                            period: 1s
                            hosts: ["${data.host}:11211"]
                    ''',
                },
            },
        )

        proc = self.start_beat()
        docker_client.images.pull('memcached:1.5.3')
        container = docker_client.containers.run('memcached:1.5.3', detach=True)

        self.wait_until(lambda: self.log_contains('Autodiscover starting runner: memcached'))

        self.wait_until(lambda: self.output_count(lambda x: x >= 1))
        container.stop()

        self.wait_until(lambda: self.log_contains('Autodiscover stopping runner: memcached'))

        output = self.read_output_json()
        proc.check_kill_and_wait()

        # Check metadata is added
        assert output[0]['docker']['container']['image'] == 'memcached:1.5.3'
        assert output[0]['docker']['container']['labels'] == {}
        assert 'name' in output[0]['docker']['container']

    @unittest.skipIf(not INTEGRATION_TESTS or
                     os.getenv("TESTING_ENVIRONMENT") == "2x",
                     "integration test not available on 2.x")
    def test_docker_labels(self):
        """
        Test docker autodiscover starts modules from labels
        """
        import docker
        docker_client = docker.from_env()

        self.render_config_template(
            autodiscover={
                'docker': {
                    'hints.enabled': 'true',
                },
            },
        )

        proc = self.start_beat()
        docker_client.images.pull('memcached:1.5.3')
        labels = {
            'co.elastic.metrics/module': 'memcached',
            'co.elastic.metrics/period': '1s',
            'co.elastic.metrics/hosts': "'${data.host}:11211'",
        }
        container = docker_client.containers.run('memcached:1.5.3', labels=labels, detach=True)

        self.wait_until(lambda: self.log_contains('Autodiscover starting runner: memcached'))

        self.wait_until(lambda: self.output_count(lambda x: x >= 1))
        container.stop()

        self.wait_until(lambda: self.log_contains('Autodiscover stopping runner: memcached'))

        output = self.read_output_json()
        proc.check_kill_and_wait()

        # Check metadata is added
        assert output[0]['docker']['container']['image'] == 'memcached:1.5.3'
        assert 'name' in output[0]['docker']['container']
