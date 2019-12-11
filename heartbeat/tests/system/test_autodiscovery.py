import os
from heartbeat import BaseTest
import unittest
import re

from beat.beat import INTEGRATION_TESTS


class TestAutodiscover(BaseTest):
    """
    Test heartbeat autodiscover
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
                          contains.docker.container.image: redis
                        config:
                          - type: tcp
                            id: myid
                            hosts: ["${data.host}:${data.port}"]
                            schedule: "@every 1s"
                            timeout: 1s
                    ''',
                },
            },
        )

        proc = self.start_beat()

        self.wait_until(lambda: self.log_contains(
            re.compile('autodiscover.+Got a start event:', re.I)))

        self.wait_until(lambda: self.output_count(lambda x: x >= 1))

        output = self.read_output_json()
        proc.check_kill_and_wait()

        matched = False
        matcher = re.compile("redis", re.I)
        for i, container in enumerate(docker_client.containers.list()):
            for tag in container.image.tags:
                if matcher.search(tag):
                    network_settings = container.attrs['NetworkSettings']
                    host = network_settings['Networks'].values()[
                        0]['IPAddress']
                    port = network_settings['Ports'].keys()[0].split("/")[0]
                    # Check metadata and docker fields are added
                    # We don't check all the docker fields because this is really the responsibility
                    # of libbeat's autodiscovery code.
                    event = output[0]
                    if event['monitor']['id'] == 'myid' and event['container']['id'] is not None:
                        matched = True

        assert matched

        self.assert_fields_are_documented(output[0])
