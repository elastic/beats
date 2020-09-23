import os
import filebeat
import unittest

from beat.beat import INTEGRATION_TESTS
from contextlib import contextmanager


class TestAutodiscover(filebeat.BaseTest):
    """
    Test filebeat autodiscover
    """
    @unittest.skipIf(not INTEGRATION_TESTS or
                     os.getenv("TESTING_ENVIRONMENT") == "2x",
                     "integration test not available on 2.x")
    def test_docker(self):
        """
        Test docker autodiscover starts input
        """
        self.render_config_template(
            inputs=False,
            autodiscover={
                'docker': {
                    'cleanup_timeout': '0s',
                    'templates': '''
                      - condition:
                          contains.docker.container.image: busybox
                        config:
                          - type: log
                            paths:
                              - %s/${data.docker.container.image}.log
                    ''' % self.working_dir,
                },
            },
        )

        self._test()

    @unittest.skipIf(not INTEGRATION_TESTS or
                     os.getenv("TESTING_ENVIRONMENT") == "2x",
                     "integration test not available on 2.x")
    def test_default_settings(self):
        """
        Test docker autodiscover default config settings
        """
        self.render_config_template(
            inputs=False,
            autodiscover={
                'docker': {
                    'cleanup_timeout': '0s',
                    'hints.enabled': 'true',
                    'hints.default_config': '''
                      type: log
                      paths:
                        - %s/${data.container.image}.log
                    ''' % self.working_dir,
                },
            },
        )

        self._test()

    def _test(self):
        image_name = 'busybox:latest'
        with open(os.path.join(self.working_dir, f'{image_name}.log'), 'wb') as f:
            f.write(b'Busybox output 1\n')

        proc = self.start_beat()
        with self.container_running(image_name):
            self.wait_until(lambda: self.log_contains('Starting runner: input'))
            self.wait_until(lambda: self.output_has(lines=1))

        output = self.read_output_json()
        proc.check_kill_and_wait()

        # Check metadata is added
        assert output[0]['message'] == 'Busybox output 1'
        assert output[0]['container']['image']['name'] == image_name
        assert output[0]['docker']['container']['labels'] == {}
        assert 'name' in output[0]['container']

        self.assert_fields_are_documented(output[0])

    @contextmanager
    def container_running(self, image_name):
        import docker
        docker_client = docker.from_env()
        container = docker_client.containers.run(image_name, 'sleep 60', detach=True, remove=True)
        yield
        container.remove(force=True)
