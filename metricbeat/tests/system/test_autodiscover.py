import os
import metricbeat
import unittest

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
                    'cleanup_timeout': '0s',
                    'templates': '''
                      - condition:
                          equals.docker.container.image: memcached:latest
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
        docker_client.images.pull('memcached:latest')
        container = docker_client.containers.run('memcached:latest', detach=True)

        self.wait_until(lambda: self.log_contains('Starting runner: RunnerGroup{memcached'))

        self.wait_until(lambda: self.output_count(lambda x: x >= 1))
        container.stop()

        self.wait_until(lambda: self.log_contains('Stopping runner: RunnerGroup{memcached'))

        output = self.read_output_json()
        proc.check_kill_and_wait()

        # Check metadata is added
        assert output[0]['container']['image']['name'] == 'memcached:latest'
        assert output[0]['docker']['container']['labels'] == {}
        assert 'name' in output[0]['container']
        self.assert_fields_are_documented(output[0])

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
                    'cleanup_timeout': '0s',
                    'hints.enabled': 'true',
                },
            },
        )

        proc = self.start_beat()
        docker_client.images.pull('memcached:latest')
        labels = {
            'co.elastic.metrics/module': 'memcached',
            'co.elastic.metrics/period': '1s',
            'co.elastic.metrics/hosts': "'${data.host}:11211'",
        }
        container = docker_client.containers.run('memcached:latest', labels=labels, detach=True)

        self.wait_until(lambda: self.log_contains('Starting runner: RunnerGroup{memcached'))

        self.wait_until(lambda: self.output_count(lambda x: x >= 1))
        container.stop()

        self.wait_until(lambda: self.log_contains('Stopping runner: RunnerGroup{memcached'))

        output = self.read_output_json()
        proc.check_kill_and_wait()

        # Check metadata is added
        assert output[0]['container']['image']['name'] == 'memcached:latest'
        assert 'name' in output[0]['container']
        self.assert_fields_are_documented(output[0])

    @unittest.skipIf(not INTEGRATION_TESTS or
                     os.getenv("TESTING_ENVIRONMENT") == "2x",
                     "integration test not available on 2.x")
    def test_config_appender(self):
        """
        Test config appenders correctly updates configs
        """
        import docker
        docker_client = docker.from_env()

        self.render_config_template(
            autodiscover={
                'docker': {
                    'cleanup_timeout': '0s',
                    'hints.enabled': 'true',
                    'appenders': '''
                      - type: config
                        condition:
                          equals.docker.container.image: memcached:latest
                        config:
                          fields:
                            foo: bar
                    ''',
                },
            },
        )

        proc = self.start_beat()
        docker_client.images.pull('memcached:latest')
        labels = {
            'co.elastic.metrics/module': 'memcached',
            'co.elastic.metrics/period': '1s',
            'co.elastic.metrics/hosts': "'${data.host}:11211'",
        }
        container = docker_client.containers.run('memcached:latest', labels=labels, detach=True)

        self.wait_until(lambda: self.log_contains('Starting runner: RunnerGroup{memcached'))

        self.wait_until(lambda: self.output_count(lambda x: x >= 1))
        container.stop()

        self.wait_until(lambda: self.log_contains('Stopping runner: RunnerGroup{memcached'))

        output = self.read_output_json()
        proc.check_kill_and_wait()

        # Check field is added
        assert output[0]['fields']['foo'] == 'bar'
        self.assert_fields_are_documented(output[0])
