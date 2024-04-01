import io
import logging
import os
import sys
import tarfile
import time

from contextlib import contextmanager


INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)

if INTEGRATION_TESTS:
    from compose.cli.command import get_project
    from compose.config.environment import Environment
    from compose.service import BuildAction
    from compose.service import ConvergenceStrategy


class ComposeMixin(object):
    """
    Manage docker-compose to ensure that needed services are running during tests
    """

    # List of required services to run INTEGRATION_TESTS
    COMPOSE_SERVICES = []

    # Additional environment variables for docker compose
    COMPOSE_ENV = {}

    # timeout waiting for health (seconds)
    COMPOSE_TIMEOUT = 300

    # add advertised host environment file
    COMPOSE_ADVERTISED_HOST = False

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
            return container.state.health.status == 'healthy' #container.inspect()['State']['Health']['Status'] == 'healthy'

        project = cls.compose_project()

        with disabled_logger('compose.service'):
            project.pull(
                ignore_pull_failures=True,
                services=cls.COMPOSE_SERVICES)

        # project.up(
        #     strategy=ConvergenceStrategy.always,
        #     service_names=cls.COMPOSE_SERVICES,
        #     timeout=30)
        project.up(
            services=cls.COMPOSE_SERVICES,
            recreate=True,
            detach=True,
        )

        # Wait for them to be healthy
        start = time.time()
        while True:
            # containers = project.containers(
            #     service_names=cls.COMPOSE_SERVICES,
            #     stopped=True)
            containers = project.ps(services=cls.COMPOSE_SERVICES, all=True)

            healthy = True
            for container in containers:
                if not container.state.status == 'running':
                    print_logs(container)
                    raise Exception(
                        "Container %s unexpectedly finished on startup" %
                        container.name)
                if not is_healthy(container):
                    healthy = False
                    break

            if healthy:
                break

            if cls.COMPOSE_ADVERTISED_HOST:
                for service in cls.COMPOSE_SERVICES:
                    cls._setup_advertised_host(project, service)

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
    def _setup_advertised_host(cls, project, service):
        """
        There are services like kafka that announce an advertised address
        to clients, who should reconnect to this address. This method
        sends the proper address to use to the container by adding a
        environment file with the SERVICE_HOST variable set to this value.
        """
        host = cls.compose_host(service=service, port=cls.COMPOSE_ADVERTISED_PORT)

        containers = project.ps(services=[service])
        for container in containers:
            container.execute(["sh", "-c", "echo SERVICE_HOST=%s >/run/compose_env" % host])
            container.execute(["sh", "-c", "chmod 644 /run/compose_env"])

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
                cls.compose_project().kill(services=cls.COMPOSE_SERVICES)

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
        #networks = list(info['NetworkSettings']['Networks'].values())
        networks = list(networks.values())
        port = port.split("/")[0]
        for network in networks:
            ip = network.ip_address
            if ip:
                return "%s:%s" % (ip, port)

    @classmethod
    def _exposed_host(cls, network_settings, port):
        """
        Return the exposed address in the host, can be used when the test is
        run from the host network. Recommended when using docker machines.
        """
        hostPort = network_settings.ports[port][0]['HostPort']
        return "localhost:%s" % hostPort

    @classmethod
    def compose_host(cls, service=None, port=None):
        if not INTEGRATION_TESTS or not cls.COMPOSE_SERVICES:
            return []

        if service is None:
            service = cls.COMPOSE_SERVICES[0]

        host_env = os.environ.get(service.upper() + "_HOST")
        if host_env:
            return host_env

        container = cls.compose_project().ps(services=[service])[0]
        # info = container.inspect()
        portsConfig = container.host_config.port_bindings  #info['HostConfig']['PortBindings']
        if len(portsConfig) == 0:
            raise Exception("No exposed ports for service %s" % service)
        if port is None:
            port = list(portsConfig.keys())[0]

        # We can use _exposed_host for all platforms when we can use host network
        # in the metricbeat container
        # networks = list(info['NetworkSettings']['Networks'].values())
        if sys.platform.startswith('linux'):
            return cls._private_host(container.network_settings.networks, port)
        return cls._exposed_host(container.network_settings, port)

    @classmethod
    def compose_project_name(cls):
        basename = os.path.basename(cls.find_compose_path())

        def positivehash(x):
            return hash(x) % ((sys.maxsize + 1) * 2)

        return "%s_%X" % (basename, positivehash(frozenset(cls.COMPOSE_ENV.items())))

    @classmethod
    def compose_project(cls):
        env = Environment(os.environ.copy())
        env.update(cls.COMPOSE_ENV)
        
        compose_files = [os.path.join(cls.find_compose_path(), "docker-compose.yml")]

        from python_on_whales import DockerClient
        docker = DockerClient(
            compose_project_name=cls.compose_project_name().lower(),
            compose_files=compose_files)
        return docker.compose
        # return get_project(cls.find_compose_path(),
        #                    project_name=cls.compose_project_name(),
        #                    environment=env)

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
