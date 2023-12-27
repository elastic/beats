import docker
import filebeat
import os
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
        with self.container_running() as container:
            self.render_config_template(
                inputs=False,
                autodiscover={
                    'docker': {
                        'cleanup_timeout': '0s',
                        'templates': f'''
                          - condition:
                                equals.docker.container.name: {container.name}
                            config:
                              - type: log
                                paths:
                                  - %s/${{data.docker.container.name}}.log
                        ''' % self.working_dir,
                    },
                },
            )

            proc = self.start_beat()
            self._test(container)

        self.wait_until(lambda: self.log_contains('Stopping runner: input'))
        proc.check_kill_and_wait()

    @unittest.skipIf(not INTEGRATION_TESTS or
                     os.getenv("TESTING_ENVIRONMENT") == "2x",
                     "integration test not available on 2.x")
    def test_default_settings(self):
        """
        Test docker autodiscover default config settings
        """
        with self.container_running() as container:
            self.render_config_template(
                inputs=False,
                autodiscover={
                    'docker': {
                        'cleanup_timeout': '0s',
                        'hints.enabled': 'true',
                        'hints.default_config': '''
                          type: log
                          paths:
                            - %s/${data.container.name}.log
                        ''' % self.working_dir,
                    },
                },
            )
            proc = self.start_beat()
            self._test(container)

        self.wait_until(lambda: self.log_contains('Stopping runner: input'))
        proc.check_kill_and_wait()

    def _test(self, container):
        with open(os.path.join(self.working_dir, f'{container.name}.log'), 'wb') as f:
            f.write(b'Busybox output 1\n')

        docker_client = docker.from_env()

        def wait_container_start():
            for i, c in enumerate(docker_client.containers.list()):
                if c.name == container.name:
                    return True

        # Ensure the container is running before checkging
        # if the input is running
        self.wait_until(
            wait_container_start,
            name="wait for test container",
            err_msg="the test container is not running yet")

        self.wait_until(lambda: self.log_contains('Starting runner: input'),
                        name="wait for input to start",
                        err_msg="did not find 'Starting runner: input' in the logs")
        self.wait_until(lambda: self.output_has(lines=1))

        output = self.read_output_json()

        # Check metadata is added
        assert output[0]['message'] == 'Busybox output 1'
        assert output[0]['container']['name'] == container.name
        assert output[0]['docker']['container']['labels'] == container.labels
        assert 'name' in output[0]['container']

        self.assert_fields_are_documented(output[0])

    @contextmanager
    def container_running(self, image_name='busybox:latest'):
        docker_client = docker.from_env()
        container = docker_client.containers.run(image_name, 'sleep 60', detach=True, remove=True)
        yield container
        container.remove(force=True)
