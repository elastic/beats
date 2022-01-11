from filebeat import BaseTest
from beat.beat import INTEGRATION_TESTS
import os
import unittest
import glob
import subprocess

import json
import logging
from parameterized import parameterized
from deepdiff import DeepDiff

# datasets for which @timestamp is removed due to date missing
remove_timestamp = {
    "activemq.audit",
    "barracuda.spamfirewall",
    "barracuda.waf",
    "bluecoat.director",
    "cef.log",
    "cisco.asa",
    "cisco.ios",
    "citrix.netscaler",
    "cylance.protect",
    "f5.bigipafm",
    "fortinet.clientendpoint",
    "haproxy.log",
    "icinga.startup",
    "imperva.securesphere",
    "infoblox.nios",
    "iptables.log",
    "juniper.junos",
    "juniper.netscreen",
    "netscout.sightline",
    "proofpoint.emailsecurity",
    "redis.log",
    "snort.log",
    "symantec.endpointprotection",
    "system.auth",
    "system.syslog",
    "crowdstrike.falcon_endpoint",
    "crowdstrike.falcon_audit",
    "zoom.webhook",
    "threatintel.otx",
    "threatintel.abuseurl",
    "threatintel.abusemalware",
    "threatintel.anomali",
    "threatintel.anomalithreatstream",
    "threatintel.malwarebazaar",
    "threatintel.recordedfuture",
    "threatintel.recordedfuture2",
    "snyk.vulnerabilities",
    "snyk.audit",
    "awsfargate.log",
}

# dataset + log file pairs for which @timestamp is kept as an exception from above
remove_timestamp_exception = {
    ('system.syslog', 'tz-offset.log'),
    ('system.auth', 'timestamp.log'),
    ('cisco.asa', 'asa.log'),
    ('cisco.asa', 'hostnames.log'),
    ('cisco.asa', 'not-ip.log'),
    ('cisco.asa', 'sample.log')
}

# array fields whose order is kept before comparison
array_fields_dont_sort = {
    "process.args"
}


def load_fileset_test_cases():
    """
    Creates a list of all modules, filesets and testfiles inside for testing.
    To execute tests for only 1 module, set the env variable TESTING_FILEBEAT_MODULES
    to the specific module name or a , separated lists of modules.
    """
    modules_dir = os.getenv("MODULES_PATH")
    if not modules_dir:
        current_dir = os.path.dirname(os.path.abspath(__file__))
        modules_dir = os.path.join(current_dir, "..", "..", "module")
    modules = os.getenv("TESTING_FILEBEAT_MODULES")
    if modules:
        modules = modules.split(",")
    else:
        modules = os.listdir(modules_dir)

    filesets_env = os.getenv("TESTING_FILEBEAT_FILESETS")

    test_cases = []

    for module in modules:
        path = os.path.join(modules_dir, module)

        if not os.path.isdir(path):
            continue

        if filesets_env:
            filesets = filesets_env.split(",")
        else:
            filesets = os.listdir(path)

        for fileset in filesets:
            if not os.path.isdir(os.path.join(path, fileset)):
                continue

            if not os.path.isfile(os.path.join(path, fileset, "manifest.yml")):
                continue

            test_files = glob.glob(os.path.join(modules_dir, module,
                                                fileset, "test", "*.log"))
            for test_file in test_files:
                test_cases.append([module, fileset, test_file])

    return test_cases


class Test(BaseTest):

    def init(self):
        self.es = self.get_elasticsearch_instance(user='admin')
        logging.getLogger("urllib3").setLevel(logging.WARNING)
        logging.getLogger("elasticsearch").setLevel(logging.ERROR)

        self.modules_path = os.path.abspath(self.working_dir +
                                            "/../../../../module")

        self.filebeat = os.path.abspath(self.working_dir +
                                        "/../../../../filebeat.test")

        self.index_name = "test-filebeat-modules"

    @parameterized.expand(load_fileset_test_cases)
    @unittest.skipIf(not INTEGRATION_TESTS,
                     "integration tests are disabled, run with INTEGRATION_TESTS=1 to enable them.")
    @unittest.skipIf(os.getenv("TESTING_ENVIRONMENT") == "2x",
                     "integration test not available on 2.x")
    def test_fileset_file(self, module, fileset, test_file):
        self.init()

        # generate a minimal configuration
        cfgfile = os.path.join(self.working_dir, "filebeat.yml")
        self.render_config_template(
            template_name="filebeat_modules",
            output=cfgfile,
            index_name=self.index_name,
            elasticsearch=self.get_elasticsearch_template_config(user='admin')
        )

        self.run_on_file(
            module=module,
            fileset=fileset,
            test_file=test_file,
            cfgfile=cfgfile)

    def run_on_file(self, module, fileset, test_file, cfgfile):
        print("Testing {}/{} on {}".format(module, fileset, test_file))

        self.assert_explicit_ecs_version_set(module, fileset)

        try:
            self.es.indices.delete_data_stream(self.index_name)
        except BaseException:
            pass
        self.wait_until(lambda: not self.es.indices.exists(self.index_name))

        cmd = [
            self.filebeat, "-systemTest",
            "-e", "-d", "*", "-once",
            "-c", cfgfile,
            "-E", "setup.ilm.enabled=false",
            "-modules={}".format(module),
            "-M", "{module}.*.enabled=false".format(module=module),
            "-M", "{module}.{fileset}.enabled=true".format(
                module=module, fileset=fileset),
            "-M", "{module}.{fileset}.var.input=file".format(
                module=module, fileset=fileset),
            "-M", "{module}.{fileset}.var.paths=[{test_file}]".format(
                module=module, fileset=fileset, test_file=test_file),
            "-M", "*.*.input.close_eof=true",
        ]
        # allow connecting older versions of Elasticsearch
        if os.getenv("TESTING_FILEBEAT_ALLOW_OLDER"):
            cmd.extend(["-E", "output.elasticsearch.allow_older_versions=true"])

        # Based on the convention that if a name contains -json the json format is needed. Currently used for LS.
        if "-json" in test_file:
            cmd.append("-M")
            cmd.append("{module}.{fileset}.var.format=json".format(
                module=module, fileset=fileset))

        output_path = os.path.join(self.working_dir)
        # Runs inside a with block to ensure file is closed afterwards
        with open(os.path.join(output_path, "output.log"), "ab") as output:
            output.write(bytes(" ".join(cmd) + "\n", "utf-8"))

            # Use a fixed timezone so results don't vary depending on the environment
            # Don't use UTC to avoid hiding that non-UTC timezones are not being converted as needed,
            # this can happen because UTC uses to be the default timezone in date parsers when no other
            # timezone is specified.
            local_env = os.environ.copy()
            local_env["TZ"] = 'Etc/GMT+2'

            subprocess.Popen(cmd,
                             env=local_env,
                             stdin=None,
                             stdout=output,
                             stderr=subprocess.STDOUT,
                             bufsize=0).wait()

        # List of errors to check in filebeat output logs
        errors = ["error loading pipeline for fileset"]
        # Checks if the output of filebeat includes errors
        contains_error, error_line = file_contains(
            os.path.join(output_path, "output.log"), errors)
        assert contains_error is False, "Error found in log:{}".format(
            error_line)

        # Make sure index exists
        self.wait_until(lambda: self.es.indices.exists(self.index_name))

        self.es.indices.refresh(index=self.index_name)
        # Loads the first 100 events to be checked
        res = self.es.search(index=self.index_name,
                             body={"query": {"match_all": {}}, "size": 100, "sort": {"log.offset": {"order": "asc"}}})
        objects = [o["_source"] for o in res["hits"]["hits"]]
        assert len(objects) > 0
        for obj in objects:
            assert obj["event"]["module"] == module, "expected event.module={} but got {}".format(
                module, obj["event"]["module"])

            # All modules must include a set processor that adds the time that
            # the event was ingested to Elasticsearch
            assert "ingested" in obj["event"], "missing event.ingested timestamp"

            assert "error" not in obj, "not error expected but got: {}.\n The related error message is: {}".format(
                obj, obj["error"].get("message"))

            if (module == "auditd" and fileset == "log") \
                    or (module == "osquery" and fileset == "result"):
                # There are dynamic fields that are not documented.
                pass
            else:
                self.assert_fields_are_documented(obj)

        self._test_expected_events(test_file, objects)

    def _test_expected_events(self, test_file, objects):

        # Generate expected files if GENERATE env variable is set
        if os.getenv("GENERATE"):
            with open(test_file + "-expected.json", 'w') as f:
                # Flatten an cleanup objects
                # This makes sure when generated on different machines / version the expected.json stays the same.
                for k, obj in enumerate(objects):
                    objects[k] = self.flatten_object(obj, {}, "")
                    clean_keys(objects[k])
                    for key in objects[k].keys():
                        if isinstance(objects[k][key], list) and key not in array_fields_dont_sort:
                            objects[k][key].sort(key=str)

                json.dump(objects, f, indent=4, separators=(
                    ',', ': '), sort_keys=True)

        with open(test_file + "-expected.json", "r") as f:
            expected = json.load(f)

        assert len(expected) == len(objects), "expected {} events to compare but got {}".format(
            len(expected), len(objects))

        # Do not perform a comparison between the resulting and expected documents
        # if the TESTING_FILEBEAT_SKIP_DIFF flag is set.
        #
        # This allows to run a basic check with older versions of ES that can lead
        # to slightly different documents without maintaining multiple sets of
        # golden files.
        if os.getenv("TESTING_FILEBEAT_SKIP_DIFF"):
            return

        for idx in range(len(expected)):
            ev = expected[idx]
            obj = objects[idx]

            # Flatten objects for easier comparing
            obj = self.flatten_object(obj, {}, "")
            clean_keys(obj)
            clean_keys(ev)

            d = DeepDiff(ev, obj, ignore_order=True)

            assert len(
                d) == 0, "The following expected object doesn't match:\n Diff:\n{}, full object: \n{}".format(d, obj)


def clean_keys(obj):
    # These keys are host dependent
    host_keys = ["agent.name", "agent.type", "agent.ephemeral_id", "agent.id"]
    # Strip host.name if event is not tagged as `forwarded`.
    if "tags" not in obj or "forwarded" not in obj["tags"]:
        host_keys.append("host.name")

    # The create timestamps area always new
    time_keys = ["event.created", "event.ingested"]
    # source path and agent.version can be different for each run
    other_keys = ["log.file.path", "agent.version"]
    # ECS versions change for any ECS release, large or small
    ecs_key = ["ecs.version"]

    # Keep source log filename for exceptions
    filename = None
    if "log.file.path" in obj:
        filename = os.path.basename(obj["log.file.path"]).lower()

    for key in host_keys + time_keys + other_keys + ecs_key:
        delete_key(obj, key)

    # Most logs from syslog need their timestamp removed because it doesn't
    # include a year.
    if obj["event.dataset"] in remove_timestamp:
        if not (obj['event.dataset'], filename) in remove_timestamp_exception:
            delete_key(obj, "@timestamp")
            # Also remove alternate time field from rsa parsers.
            delete_key(obj, "rsa.time.event_time")
        else:
            # excluded events need to have their filename saved to the expected.json
            # so that the exception mechanism can be triggered when the json is
            # loaded.
            obj["log.file.path"] = filename

    # Remove @timestamp from aws vpc flow log with custom format (with no event.end time).
    if obj["event.dataset"] == "aws.vpcflow":
        if "event.end" not in obj:
            delete_key(obj, "@timestamp")


def delete_key(obj, key):
    if key in obj:
        del obj[key]


def file_contains(filepath, strings):
    with open(filepath, 'r') as file:
        for line in file:
            for string in strings:
                if string in line:
                    return True, line
    return False, None


def pretty_json(obj):
    return json.dumps(obj, indent=2, separators=(',', ': '))
