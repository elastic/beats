"""
Main python for metricbeat tests
"""
import logging
import os
import re
import sys
import yaml
from beat.beat import TestCase
from beat.tags import tag
from parameterized import parameterized_class

COMMON_FIELDS = ["@timestamp", "agent", "metricset.name", "metricset.host",
                 "metricset.module", "metricset.rtt", "host.name", "service.name", "event", "ecs"]

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)

logging.getLogger("urllib3").setLevel(logging.WARNING)


class BaseTest(TestCase):
    """
    BaseTest implements the individual test class for metricbeat
    """

    @classmethod
    def setUpClass(cls):  # pylint: disable=invalid-name
        """
        initializes the Test class
        """
        if not hasattr(cls, 'beat_name'):
            cls.beat_name = "metricbeat"

        if not hasattr(cls, 'beat_path'):
            cls.beat_path = os.path.abspath(
                os.path.join(os.path.dirname(__file__), "../../"))

        super().setUpClass()

    def de_dot(self, existing_fields):
        """
        de_dot strips the dot notation from events
        """
        fields = {}

        # Dedot first level of dots
        for key in existing_fields:
            parts = key.split('.', 1)

            if len(parts) > 1:
                if parts[0] not in fields:
                    fields[parts[0]] = {}

                fields[parts[0]][parts[1]] = parts[1]
            else:
                fields[parts[0]] = parts[0]

        # Dedot further levels recursively
        for key in fields:
            if isinstance(fields[key], dict):
                fields[key] = self.de_dot(fields[key])

        return fields

    def assert_no_logged_warnings(self, replace=None):
        """
        Assert that the log file contains no ERROR or WARN lines.
        """
        log = self.get_log()

        pattern = self.build_log_regex(r"\[cfgwarn\]")
        log = pattern.sub("", log)

        # Jenkins runs as a Windows service and when Jenkins executes these
        # tests the Beat is confused since it thinks it is running as a service.
        pattern = self.build_log_regex(
            "The service process could not connect to the service controller.")
        log = pattern.sub("", log)

        if replace:
            for r in replace:
                pattern = self.build_log_regex(r)
                log = pattern.sub("", log)
        self.assertNotRegex(log, "\tERROR\t|\tWARN\t")

    def build_log_regex(self, message):
        return re.compile(r"^.*\t(?:ERROR|WARN)\t.*" + message + r".*$", re.MULTILINE)

    def check_metricset(self, module, metricset, hosts, fields=[], extras=[]):
        """
        Method to test a metricset for its fields
        """
        self.render_config_template(modules=[{
            "name": module,
            "metricsets": [metricset],
            "hosts": hosts,
            "period": "1s",
            "extras": extras,
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        print(output)
        self.assertTrue(len(output) >= 1)
        evt = output[0]
        print(evt)

        fields = COMMON_FIELDS + fields
        print(fields)
        self.assertCountEqual(self.de_dot(fields), evt.keys())

        self.assert_fields_are_documented(evt)


def supported_versions(path):
    """
    Returns variants information as expected by parameterized_class,
    that is as a list of lists with an only element that is used to
    override the value of COMPOSE_ENV.
    """
    if not os.path.exists(path):
        # Return an empty variant so a class is instantiated with defaults
        return [[{}]]

    variants = []
    with open(path) as f:
        versions_info = yaml.safe_load(f)

        for variant in versions_info['variants']:
            variants += [[variant]]

    return variants


def parameterized_with_supported_versions(base_class):
    """
    Decorates a class so instead of the base class, multiple copies
    of it are registered, one for each supported version.
    """
    class_dir = os.path.abspath(os.path.dirname(
        sys.modules[base_class.__module__].__file__))
    versions_path = os.path.join(class_dir, '_meta', 'supported-versions.yml')
    variants = supported_versions(versions_path)
    decorator = parameterized_class(['COMPOSE_ENV'], variants)
    decorator(base_class)
