import io
import logging
import os
import sys
import tarfile
import time
import tempfile
import random
from pathlib import Path

from contextlib import contextmanager
from python_on_whales import DockerClient

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class ComposeMixin(object):
    """
    Manage docker-compose to ensure that needed services are running during tests
    """

    # List of required services to run INTEGRATION_TESTS
    COMPOSE_SERVICES = []

    # Additional environment variables for docker compose
    COMPOSE_ENV = {}

    # timeout waiting for health (seconds)
    COMPOSE_TIMEOUT = 1

    # add advertised host environment file
    COMPOSE_ADVERTISED_HOST = False

    # max retries to check if services are healthy
    COMPOSE_MAX_RETRIES = 7

    # port to advertise when COMPOSE_ADVERTISED_HOST is set to true
    COMPOSE_ADVERTISED_PORT = None

    @classmethod
    def compose_up(cls):
        """
        Ensure *only* the services defined under `COMPOSE_SERVICES` are running and healthy
        """
        if not INTEGRATION_TESTS or not cls.COMPOSE_SERVICES:
            return

        if os.environ.get('NO_COMPOSE'):
            return

        def print_logs(container):
            print("---- " + container.name)
            print(container.logs())
            print("----")

        def is_healthy(container):
            print("Checking health of %s and the status is: %s" % (container.name, container.state.status))
            return container.state.status == 'running'

        project = cls.compose_project()

        with disabled_logger('compose.service'):
            project.pull(
                ignore_pull_failures=True,
                services=cls.COMPOSE_SERVICES)

        project.up(
            services=cls.COMPOSE_SERVICES,
            recreate=True,
            detach=True,
            remove_orphans=True,
            color=False,
        )

        print("Docker-compose services: %s" % ','.join(cls.COMPOSE_SERVICES))
        print("Docker compose advertised host: %s" % cls.COMPOSE_ADVERTISED_HOST)

        containers = project.ps(services=cls.COMPOSE_SERVICES, all=True)
        retry_delay = cls.COMPOSE_TIMEOUT
        for attempt in range(cls.COMPOSE_MAX_RETRIES):
            print("Checking health status: %s with delay of %s" % (attempt, retry_delay))
            healthy = True
            for container in containers:
                if not is_healthy(container):
                    healthy = False
                    break

            if healthy:
                break

            if cls.COMPOSE_ADVERTISED_HOST:
                for service in cls.COMPOSE_SERVICES:
                    cls._setup_advertised_host(project, service)

            time.sleep(retry_delay)
            retry_delay *= 2
            retry_delay += random.uniform(0, 1)
        else:
            # This part executes if the loop completes without a 'break'
            raise Exception("Max retries reached without achieving health status")

    @classmethod
    def _setup_advertised_host(cls, project, service):
        """
        There are services like kafka that announce an advertised address
        to clients, who should reconnect to this address. This method
        sends the proper address to use to the container by adding a
        environment file with the SERVICE_HOST variable set to this value.
        """
        host = cls.compose_host(service=service, port=cls.COMPOSE_ADVERTISED_PORT)

        content = "SERVICE_HOST=%s" % host
        info = tarfile.TarInfo(name="/run/compose_env")
        info.mode = 0o100644
        info.size = len(content)

        data = io.BytesIO()
        tar = tarfile.TarFile(fileobj=data, mode='w')
        tar.addfile(info, fileobj=io.BytesIO(content.encode("utf-8")))
        tar.close()

        containers = project.ps(service_names=[service])
        for container in containers:
            container.put_archive(path="/", data=data.getvalue())

    @classmethod
    def compose_down(cls):
        """
        Stop all running containers
        """
        if os.environ.get('NO_COMPOSE'):
            return

        if INTEGRATION_TESTS and cls.COMPOSE_SERVICES:
            # Use down on per-module scenarios to release network pools too
            if os.path.basename(os.path.dirname(cls.find_compose_path())) == "module":
                cls.compose_project().down(volumes=True)
            else:
                for service in cls.COMPOSE_SERVICES:
                    cls.compose_project().stop(services=[service])

    @classmethod
    def get_hosts(cls):
        return [cls.compose_host()]

    @classmethod
    def _private_host(cls, networks, port):
        """
        Return the address of the container, it should be reachable from the
        host if docker is being run natively. To be used when the tests are
        run from another container in the same network. It also works when
        running from the host network if the docker daemon runs natively.
        """
        networks = list(networks.values())
        port = port.split("/")[0]
        for network in networks:
            ip = network.ip_address
            if ip:
                return "%s:%s" % (ip, port)

    @classmethod
    def _exposed_host(cls, info, port):
        """
        Return the exposed address in the host, can be used when the test is
        run from the host network. Recommended when using docker machines.
        """
        host_port = info.network_settings.ports[port][0]['HostPort']
        return "localhost:%s" % host_port

    @classmethod
    def compose_host(cls, service=None, port=None):
        if not INTEGRATION_TESTS or not cls.COMPOSE_SERVICES:
            return []

        if service is None:
            service = cls.COMPOSE_SERVICES[0]

        host_env = os.environ.get(service.upper() + "_HOST")
        if host_env:
            return host_env

        containers = cls.compose_project().ps(services=[service], all=False)
        try:
            container = containers[0]
        except IndexError:
            raise Exception(f"No container found for service {service}")

        try:
            ports_config = container.host_config.port_bindings
            if not ports_config:
                raise Exception(f"No exposed ports for service {service}")
            if port is None:
                port = list(ports_config.keys())[0]
        except (IndexError, KeyError):
            raise Exception(f"No valid port binding found for service {service}")

        if sys.platform.startswith('linux'):
            return cls._private_host(container.network_settings.networks, port)
        return cls._exposed_host(container._get_inspect_result(), port)

    @classmethod
    def compose_project_name(cls):
        basename = os.path.basename(cls.find_compose_path())

        def positivehash(x):
            return hash(x) % ((sys.maxsize + 1) * 2)

        return "%s_%X" % (basename, positivehash(frozenset(cls.COMPOSE_ENV.items())))

    @classmethod
    def compose_project(cls):
        env = os.environ.copy()
        env.update(cls.COMPOSE_ENV)

        compose_files = [os.path.join(cls.find_compose_path(), "docker-compose.yml")]

        with tempfile.NamedTemporaryFile(mode='w+', delete=False) as env_file:
            # Write variables from os.environ
            for key, value in os.environ.items():
                env_file.write(f"{key}={value}\n")

            # Write variables from COMPOSE_ENV
            for key, value in cls.COMPOSE_ENV.items():
                env_file.write(f"{key}={value}\n")

            env_file_path = Path(env_file.name)

        docker_client = DockerClient(
            compose_project_name=cls.compose_project_name().lower(),
            compose_files=compose_files,
            compose_env_file=env_file_path,
        )

        return docker_client.compose

    @classmethod
    def find_compose_path(cls):
        class_dir = os.path.abspath(os.path.dirname(sys.modules[cls.__module__].__file__))
        while True:
            if os.path.exists(os.path.join(class_dir, "docker-compose.yml")):
                return class_dir
            class_dir, current = os.path.split(class_dir)
            if current == '':  # We have reached root
                raise Exception("failed to find a docker-compose.yml file")

    @classmethod
    def get_service_log(cls, service):
        container = cls.compose_project().ps(services=[service])[0]
        return container.logs()

    @classmethod
    def service_log_contains(cls, service, msg):
        log = cls.get_service_log(service)
        counter = 0
        for line in log.splitlines():
            if line.find(msg) >= 0:
                counter += 1
        return counter > 0


@contextmanager
def disabled_logger(name):
    logger = logging.getLogger(name)
    old_level = logger.getEffectiveLevel()
    logger.setLevel(logging.CRITICAL)
    try:
        yield logger
    finally:
        logger.setLevel(old_level)
