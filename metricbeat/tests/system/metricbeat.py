"""
Main python for metricbeat tests
"""
import logging
import os
import re
import sys
import yaml
from beat.beat import TestCase  # pylint: disable=import-error
from parameterized import parameterized_class  # pylint: disable=import-error

COMMON_FIELDS = ["@timestamp", "agent", "metricset.name", "metricset.host",
                 "metricset.module", "metricset.rtt", "host.name", "service.name", "event", "ecs"]

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)

logging.getLogger("urllib3").setLevel(logging.WARNING)

P_WIN = "win32"
P_LINUX = "linux"
P_DARWIN = "darwin"
P_DEF = "default"


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
        for key, val in fields.items():
            if isinstance(val, dict):
                fields[key] = self.de_dot(fields[key])

        return fields

    def assert_fields_for_platform(self, good_fields: object, written_fields: object):
        """
        Assert that the event contains a given set of fields depending on the OS
        """
        if sys.platform not in good_fields:
            self.assertCountEqual(self.de_dot(
                good_fields[P_DEF]), written_fields.keys())
            return

        self.assertCountEqual(self.de_dot(
            good_fields[sys.platform]), written_fields.keys())

    def assert_no_logged_warnings(self, replace=None):
        """
        Assert that the log file contains no ERROR or WARN lines.
        """
        log = self.get_log()

        pattern = build_log_regex(r"\[cfgwarn\]")
        log = pattern.sub("", log)

        # Jenkins runs as a Windows service and when Jenkins executes these
        # tests the Beat is confused since it thinks it is running as a service.
        pattern = build_log_regex(
            "The service process could not connect to the service controller.")
        log = pattern.sub("", log)

        if replace:
            for rep in replace:
                pattern = build_log_regex(rep)
                log = pattern.sub("", log)
        self.assertNotRegex(log, "\tERROR\t|\tWARN\t")

    def run_beat_and_stop(self):
        """
        starts and runs metricbeat based for a child unit test
        """
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

    def check_metricset(self, module, metricset, hosts, fields: list = None, extras: list = None):
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
    with open(path, encoding="utf-8") as file:
        versions_info = yaml.safe_load(file)

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


def build_log_regex(message):
    """
    build_log_regex returns compiled regex for ERROR/WARN messages in logs
    """
    return re.compile(r"^.*\t(?:ERROR|WARN)\t.*" + message + r".*$", re.MULTILINE)
