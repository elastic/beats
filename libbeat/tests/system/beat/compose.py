from __future__ import absolute_import
import os
import time


INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


if INTEGRATION_TESTS:
    from compose.cli.command import get_project
    from compose.service import BuildAction


class ComposeMixin(object):
    """
    Manage docker-compose to ensure that needed services are running during tests
    """

    # List of required services to run INTEGRATION_TESTS
    COMPOSE_SERVICES = []

    # docker-compose.yml dir path
    COMPOSE_PROJECT_DIR = '.'

    # timeout waiting for health (seconds)
    COMPOSE_TIMEOUT = 60

    @classmethod
    def compose_up(cls):
        """
        Ensure *only* the services defined under `COMPOSE_SERVICES` are running and healthy
        """
        if INTEGRATION_TESTS and cls.COMPOSE_SERVICES:
            cls.compose_project().up(
                service_names=cls.COMPOSE_SERVICES,
                do_build=BuildAction.force,
                timeout=30)

            # Wait for them to be healthy
            healthy = False
            seconds = cls.COMPOSE_TIMEOUT
            while not healthy and seconds > 0:
                print("Seconds: %d".format(seconds))
                seconds -= 1
                time.sleep(1)
                healthy = True
                for container in cls.compose_project().containers(service_names=cls.COMPOSE_SERVICES):
                    if container.inspect()['State']['Health']['Status'] != 'healthy':
                        healthy = False
                        break

            if not healthy:
                raise Exception('Timeout while waiting for healthy docker-compose services')

    @classmethod
    def compose_down(cls):
        """
        Stop all running containers
        """
        if INTEGRATION_TESTS and cls.COMPOSE_SERVICES:
            cls.compose_project().kill(service_names=cls.COMPOSE_SERVICES)

    @classmethod
    def compose_project(cls):
        return get_project(cls.COMPOSE_PROJECT_DIR, project_name=os.environ.get('DOCKER_COMPOSE_PROJECT_NAME'))
