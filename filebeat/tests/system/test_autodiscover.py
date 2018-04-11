import os
import filebeat
import unittest

from beat.beat import INTEGRATION_TESTS


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
        import docker
        docker_client = docker.from_env()

        self.render_config_template(
            inputs=False,
            autodiscover={
                'docker': {
                    'templates': '''
                      - condition:
                          equals.docker.container.image: busybox
                        config:
                          - type: log
                            paths:
                              - %s/${data.docker.container.image}.log
                    ''' % self.working_dir,
                },
            },
        )

        with open(os.path.join(self.working_dir, 'busybox.log'), 'wb') as f:
            f.write('Busybox output 1\n')

        proc = self.start_beat()
        docker_client.images.pull('busybox')
        docker_client.containers.run('busybox', 'sleep 1')

        self.wait_until(lambda: self.log_contains('Autodiscover starting runner: input'))
        self.wait_until(lambda: self.log_contains('Autodiscover stopping runner: input'))

        output = self.read_output_json()
        proc.check_kill_and_wait()

        # Check metadata is added
        assert output[0]['message'] == 'Busybox output 1'
        assert output[0]['docker']['container']['image'] == 'busybox'
        assert output[0]['docker']['container']['labels'] == {}
        assert 'name' in output[0]['docker']['container']
