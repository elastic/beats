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
        if not INTEGRATION_TESTS or not cls.COMPOSE_SERVICES:
            return

        def print_logs(container):
            print("---- " + container.name_without_project)
            print(container.logs())
            print("----")

        def is_healthy(container):
            return container.inspect()['State']['Health']['Status'] == 'healthy'

        project = cls.compose_project()
        project.up(
            service_names=cls.COMPOSE_SERVICES,
            do_build=BuildAction.force,
            timeout=30)

        # Wait for them to be healthy
        start = time.time()
        while True:
            containers = project.containers(
                service_names=cls.COMPOSE_SERVICES,
                stopped=True)

            healthy = True
            for container in containers:
                if not container.is_running:
                    print_logs(container)
                    raise Exception(
                        "Container %s unexpectedly finished on startup" %
                        container.name_without_project)
                if not is_healthy(container):
                    healthy = False
                    break

            if healthy:
                break

            time.sleep(1)
            timeout = time.time() - start > cls.COMPOSE_TIMEOUT
            if timeout:
                for container in containers:
                    if not is_healthy(container):
                        print_logs(container)
                raise Exception(
                    "Timeout while waiting for healthy "
                    "docker-compose services: %s" %
                    ','.join(cls.COMPOSE_SERVICES))

    @classmethod
    def compose_down(cls):
        """
        Stop all running containers
        """
        if INTEGRATION_TESTS and cls.COMPOSE_SERVICES:
            cls.compose_project().kill(service_names=cls.COMPOSE_SERVICES)

    @classmethod
    def compose_hosts(cls):
        if not INTEGRATION_TESTS or not cls.COMPOSE_SERVICES:
            return []

        hosts = []
        for container in cls.compose_project().containers(service_names=cls.COMPOSE_SERVICES):
            network_settings = container.inspect()['NetworkSettings']
            for network in network_settings['Networks'].values():
                if network['IPAddress']:
                    hosts.append(network['IPAddress'])
        return hosts

    @classmethod
    def compose_project(cls):
        return get_project(cls.COMPOSE_PROJECT_DIR, project_name=os.environ.get('DOCKER_COMPOSE_PROJECT_NAME'))
