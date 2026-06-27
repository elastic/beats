import io
import os
import subprocess
import sys
import tarfile
import time

import docker as docker_sdk


INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class ComposeMixin(object):
    """
    Manage docker compose to ensure that needed services are running during tests
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
    def _compose_cmd(cls, *args):
        """Build and return a docker compose command list and its environment."""
        compose_path = cls.find_compose_path()
        project_name = cls.compose_project_name()
        cmd = [
            "docker", "compose",
            "-p", project_name,
            "-f", os.path.join(compose_path, "docker-compose.yml"),
        ]
        cmd.extend(args)
        env = os.environ.copy()
        env.update(cls.COMPOSE_ENV)
        return cmd, env

    @classmethod
    def _run_compose(cls, *args, **kwargs):
        """Run a docker compose command."""
        cmd, env = cls._compose_cmd(*args)
        return subprocess.run(cmd, env=env, **kwargs)

    @classmethod
    def _get_project_containers(cls, service_names=None, include_stopped=False):
        """Get containers for the project using Docker SDK."""
        client = docker_sdk.from_env()
        project_name = cls.compose_project_name()

        filters = {
            "label": ["com.docker.compose.project=%s" % project_name]
        }

        containers = client.containers.list(all=include_stopped, filters=filters)

        if service_names:
            containers = [
                c for c in containers
                if c.labels.get("com.docker.compose.service") in service_names
            ]

        return containers

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
            service = container.labels.get("com.docker.compose.service", container.name)
            print("---- " + service)
            print(container.logs().decode('utf-8', errors='replace'))
            print("----")

        def is_healthy(container):
            container.reload()
            health = container.attrs.get('State', {}).get('Health', {})
            return health.get('Status') == 'healthy'

        # Pull images (ignore failures)
        cls._run_compose(
            "pull", *cls.COMPOSE_SERVICES,
            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

        # Start services
        cls._run_compose(
            "up", "-d", "--force-recreate", "--timeout", "30",
            *cls.COMPOSE_SERVICES,
            check=True)

        # Wait for them to be healthy
        start = time.time()
        while True:
            containers = cls._get_project_containers(
                service_names=cls.COMPOSE_SERVICES,
                include_stopped=True)

            healthy = True
            for container in containers:
                container.reload()
                if container.status != 'running':
                    print_logs(container)
                    raise Exception(
                        "Container %s unexpectedly finished on startup" %
                        container.labels.get("com.docker.compose.service", container.name))
                if not is_healthy(container):
                    healthy = False
                    break

            if healthy:
                break

            if cls.COMPOSE_ADVERTISED_HOST:
                for service in cls.COMPOSE_SERVICES:
                    cls._setup_advertised_host(service)

            time.sleep(1)
            timeout = time.time() - start > cls.COMPOSE_TIMEOUT
            if timeout:
                for container in containers:
                    if not is_healthy(container):
                        print_logs(container)
                raise Exception(
                    "Timeout while waiting for healthy "
                    "docker compose services: %s" %
                    ','.join(cls.COMPOSE_SERVICES))

    @classmethod
    def _setup_advertised_host(cls, service):
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

        containers = cls._get_project_containers(service_names=[service])
        for container in containers:
            container.put_archive("/", data.getvalue())

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
                cls._run_compose("down", "-v")
            else:
                cls._run_compose("kill", *cls.COMPOSE_SERVICES)

    @classmethod
    def get_hosts(cls):
        return [cls.compose_host()]

    @classmethod
    def _private_host(cls, info, port):
        """
        Return the address of the container, it should be reachable from the
        host if docker is being run natively. To be used when the tests are
        run from another container in the same network. It also works when
        running from the host network if the docker daemon runs natively.
        """
        networks = list(info['NetworkSettings']['Networks'].values())
        port = port.split("/")[0]
        for network in networks:
            ip = network['IPAddress']
            if ip:
                return "%s:%s" % (ip, port)

    @classmethod
    def _exposed_host(cls, info, port):
        """
        Return the exposed address in the host, can be used when the test is
        run from the host network. Recommended when using docker machines.
        """
        hostPort = info['NetworkSettings']['Ports'][port][0]['HostPort']
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

        containers = cls._get_project_containers(service_names=[service])
        if not containers:
            raise Exception("No containers for service %s" % service)

        container = containers[0]
        container.reload()
        info = container.attrs
        portsConfig = info['HostConfig']['PortBindings']
        if len(portsConfig) == 0:
            raise Exception("No exposed ports for service %s" % service)
        if port is None:
            port = list(portsConfig.keys())[0]

        # We can use _exposed_host for all platforms when we can use host network
        # in the metricbeat container
        if sys.platform.startswith('linux'):
            return cls._private_host(info, port)
        return cls._exposed_host(info, port)

    @classmethod
    def compose_project_name(cls):
        basename = os.path.basename(cls.find_compose_path())

        def positivehash(x):
            return hash(x) % ((sys.maxsize + 1) * 2)

        # Docker Compose V2 requires project names to be lowercase.
        return ("%s_%x" % (basename, positivehash(frozenset(cls.COMPOSE_ENV.items())))).lower()

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
        containers = cls._get_project_containers(service_names=[service])
        if not containers:
            return b''
        return containers[0].logs()

    @classmethod
    def service_log_contains(cls, service, msg):
        log = cls.get_service_log(service)
        counter = 0
        for line in log.splitlines():
            if line.find(msg.encode("utf-8")) >= 0:
                counter += 1
        return counter > 0
