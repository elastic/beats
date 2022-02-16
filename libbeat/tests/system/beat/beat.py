"""
beat defines the basic testing infrastructure used by individual python unit/integration tests
"""

import subprocess

import unittest
import os
import shutil
import json
import signal
import sys
import time
import hashlib
import re
import glob
from datetime import datetime, timedelta

import jinja2
import yaml
from elasticsearch import Elasticsearch

from .compose import ComposeMixin


BEAT_REQUIRED_FIELDS = ["@timestamp",
                        "agent.type", "agent.name", "agent.version"]

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)

YAML_CACHE = {}

REGEXP_TYPE = type(re.compile("t"))


def json_raise_on_duplicates(ordered_pairs):
    """
    Helper function to reject duplicate keys. To be used as a custom hook in JSON unmarshaling
    to error out in case of any duplicates in the keys.
    """
    key_dict = {}
    for key, val in ordered_pairs:
        if key in key_dict:
            raise ValueError(f"duplicate key: {key}")
        key_dict[key] = val
    return key_dict


def get_elasticsearch_url():
    """
    Returns a string with the Elasticsearch URL
    """
    return "http://{host}:{port}".format(
        host=os.getenv("ES_HOST", "localhost"),
        port=os.getenv("ES_PORT", "9200"),
    )


def get_elasticsearch_url_ssl():
    """
    Returns a string with the Elasticsearch URL
    """
    return "https://{host}:{port}".format(
        host=os.getenv("ES_HOST_SSL", "localhost"),
        port=os.getenv("ES_PORT_SSL", "9205"),
    )


def get_kibana_url():
    """
    Returns kibana host URL
    """
    return "http://{host}:{port}".format(
        host=os.getenv("KIBANA_HOST", "localhost"),
        port=os.getenv("KIBANA_PORT", "5601"),
    )


def get_elasticsearch_instance(security=True, ssl=False, url=None, user=None):
    """
    Returns an elasticsearch.Elasticsearch instance built from the
    env variables like the integration tests.
    """
    if url is None:
        if ssl:
            url = get_elasticsearch_url_ssl()
        else:
            url = get_elasticsearch_url()

    if security:
        username = user or os.getenv("ES_USER", "")
        password = os.getenv("ES_PASS", "")
        es_instance = Elasticsearch([url], http_auth=(username, password))
    else:
        es_instance = Elasticsearch([url])
    return es_instance


def get_kibana_template_config(security=True, user=None):
    """
    Returns a Kibana template suitable for a Beat
    """
    template = {
        "host": get_kibana_url()
    }

    if security:
        template["user"] = user or os.getenv("ES_USER", "")
        template["pass"] = os.getenv("ES_PASS", "")

    return template


def get_elasticsearch_template_config(self, security=True, user=None):
    """
    Returns a template suitable for a Beats config
    """
    template = {
        "host": get_elasticsearch_url(),
    }

    if security:
        template["user"] = user or os.getenv("ES_USER", "")
        template["pass"] = os.getenv("ES_PASS", "")

    return template


class WaitTimeoutError(Exception):
    """
    WaitTimeoutError is raised by the wait_until function if the `until` logic passes its timeout.
    """
    pass


class Proc():
    """
    Slim wrapper on subprocess.Popen that redirects
    both stdout and stderr to a file on disk and makes
    sure to stop the process and close the output file when
    the object gets collected.
    """

    def __init__(self, args, outputfile, env=None):
        self.args = args
        self.output = open(outputfile, "ab")
        self.stdin_read, self.stdin_write = os.pipe()
        if env:
            self.env = env
        else:
            self.env = {}

        self.proc = None

    def start(self):
        """
        start wraps the underlying `popen` method used by the tests
        """
        # ensure that the environment is inherited to the subprocess.
        variables = os.environ.copy()
        variables.update(self.env)

        if sys.platform.startswith("win"):
            self.proc = subprocess.Popen(
                self.args,
                stdin=self.stdin_read,
                stdout=self.output,
                stderr=subprocess.STDOUT,
                bufsize=0,
                creationflags=subprocess.CREATE_NEW_PROCESS_GROUP,
                env=variables)
        else:
            self.proc = subprocess.Popen(
                self.args,
                stdin=self.stdin_read,
                stdout=self.output,
                stderr=subprocess.STDOUT,
                bufsize=0,
                env=variables)
            # If a "No such file or directory" error points you here, run
            # "make metricbeat.test" on metricbeat folder
        return self.proc

    def kill(self):
        """
        kill terminates the process started by `Popen`
        """
        if sys.platform.startswith("win"):
            # proc.terminate on Windows does not initiate a graceful shutdown
            # through the processes signal handlers it just kills it hard. So
            # this sends a SIGBREAK. You cannot sends a SIGINT (CTRL_C_EVENT)
            # to a process group in Windows, otherwise Ctrl+C would be
            # sent.
            self.proc.send_signal(
                signal.CTRL_BREAK_EVENT)  # pylint: disable=no-member
        else:
            self.proc.terminate()

    def wait(self):
        """
        wait wraps the underlying `Popen` wait call, and will wait for the process to exit
        """
        try:
            return self.proc.wait()
        finally:
            self.output.close()

    def check_wait(self, exit_code=0):
        """
        check_wait waits for the process to exit, and checks the return code of the process
        """
        actual_exit_code = self.wait()
        assert actual_exit_code == exit_code, f"Expected exit code to be {exit_code}, but it was {actual_exit_code}"
        return actual_exit_code

    def kill_and_wait(self):
        """
        kill_and_wait will kill the process and wait for it to return
        """
        self.kill()
        os.close(self.stdin_write)
        return self.wait()

    def check_kill_and_wait(self, exit_code=0):
        """
        check_kill_and_wait will kill the process, then check the resulting exit code
        """
        self.kill()
        os.close(self.stdin_write)
        return self.check_wait(exit_code=exit_code)

    def __del__(self):
        # Ensure the process is stopped.
        try:
            self.proc.terminate()
            self.proc.kill()
        except BaseException:
            pass
        # Ensure the output is closed.
        try:
            self.output.close()
        except BaseException:
            pass


class TestCase(unittest.TestCase, ComposeMixin):
    """
    TestCase is the class for individual tests, and provides the methods for starting the beat and checking output
    """
    today = datetime.now().strftime("%Y%m%d")
    default_docker_args = ["-e", "-v", "-d", "*"]

    @classmethod
    def setUpClass(cls):

        # Path to test binary
        if not hasattr(cls, 'beat_name'):
            cls.beat_name = "beat"

        if not hasattr(cls, 'beat_path'):
            cls.beat_path = "."

        # Path to test binary
        if not hasattr(cls, 'test_binary'):
            cls.test_binary = os.path.abspath(
                cls.beat_path + "/" + cls.beat_name + ".test")

        if not hasattr(cls, 'template_paths'):
            cls.template_paths = [
                cls.beat_path,
                os.path.abspath(os.path.join(cls.beat_path, "../libbeat"))
            ]

        # Create build path
        build_dir = cls.beat_path + "/build"
        cls.build_path = build_dir + "/system-tests/"

        # Start the containers needed to run these tests
        cls.compose_up_with_retries()

    @classmethod
    def tearDownClass(cls):
        cls.compose_down()

    @classmethod
    def compose_up_with_retries(cls):
        """
        compose_up_with_retries runs docker compose to start the test containers
        """
        retries = 3
        for i in range(retries):
            try:
                cls.compose_up()
                return
            except Exception as ex:
                if i + 1 >= retries:
                    raise ex
                print(f"Compose up failed, retrying: {ex}")
                cls.compose_down()

    def run_beat(self,
                 cmd=None,
                 config=None,
                 output=None,
                 logging_args: list = None,
                 extra_args: list = None,
                 exit_code=None,
                 env: object = None):
        """
        Executes beat.
        Waits for the process to finish before returning to
        the caller.
        """
        proc = self.start_beat(cmd=cmd, config=config, output=output,
                               logging_args=logging_args,
                               extra_args=extra_args, env=env)
        if exit_code is not None:
            return proc.check_wait(exit_code)

        return proc.wait()

    def start_beat(self,
                   cmd=None,
                   config=None,
                   output=None,
                   logging_args: list = None,
                   extra_args: list = None,
                   env: object = None,
                   home=""):
        """
        Starts beat and returns the process handle. The
        caller is responsible for stopping / waiting for the
        Proc instance.
        """
        if logging_args is None:
            logging_args = self.default_docker_args
        # Init defaults
        if cmd is None:
            cmd = self.test_binary

        if config is None:
            config = self.beat_name + ".yml"

        if output is None:
            output = self.beat_name + "-" + self.today + ".ndjson"

        args = [cmd, "-systemTest"]
        if os.getenv("TEST_COVERAGE") == "true":
            args += [
                "-test.coverprofile",
                os.path.join(self.working_dir, "coverage.cov"),
            ]

        path_home = os.path.normpath(self.working_dir)
        if home:
            path_home = home

        args += [
            "-path.home", path_home,
            "-c", os.path.join(self.working_dir, config),
        ]

        if logging_args:
            args.extend(logging_args)

        if extra_args is not None:
            args.extend(extra_args)

        proc = Proc(args, os.path.join(self.working_dir, output), env)
        proc.start()
        return proc

    def render_config_template(self, template_name=None,
                               output=None, **kargs):
        """
        render_config_template fetches a given jinja2 config template and writes the formatted config
        """

        # Init defaults
        if template_name is None:
            template_name = self.beat_name

        template_path = "./tests/system/config/" + template_name + ".yml.j2"

        if output is None:
            output = self.beat_name + ".yml"

        template = self.template_env.get_template(template_path)

        kargs["beat"] = self
        output_str = template.render(**kargs)

        output_path = os.path.join(self.working_dir, output)
        with open(output_path, "wb") as beat_output:
            os.chmod(output_path, 0o600)
            beat_output.write(output_str.encode('utf_8'))

    def default_output_file(self):
        """
        default_output_file returns the default path and name of the beat metrics file output
        """
        return "output/" + self.beat_name + "-" + self.today + ".ndjson"

    def read_output(self,
                    output_file=None,
                    required_fields=None,
                    filter_key: str = ""):
        """
        read_output Returns output as JSON object with flattened fields (. notation)
        """

        # Init defaults
        if output_file is None:
            output_file = self.default_output_file()

        jsons = []
        with open(os.path.join(self.working_dir, output_file), "r", encoding="utf_8") as beat_output:
            for line in beat_output:
                if len(line) == 0 or line[len(line) - 1] != "\n":
                    # hit EOF
                    break

                try:
                    jsons.append(self.flatten_object(json.loads(
                        line, object_pairs_hook=json_raise_on_duplicates), []))
                except BaseException:
                    print(f"Failed to load the json {line}")
                    raise

        self.all_have_fields(jsons, required_fields or BEAT_REQUIRED_FIELDS)
        if filter_key != "":
            return list(filter(lambda x: filter_key in x, jsons))
        return jsons

    def read_output_filter(self, key: str, output_file=None, required_fields=None):
        """
        same as read_output, but filters the events down based on the availability of a key
        this is needed with newer versions of the system module will only report fields if they contain valid data.
        """
        output = self.read_output(
            output_file=output_file, required_fields=required_fields)

        return list(filter(lambda x: key in x, output))

    def read_output_json(self, output_file=None):
        """
        read_output_json Returns output as JSON object
        """

        # Init defaults
        if output_file is None:
            output_file = self.default_output_file()

        jsons = []
        with open(os.path.join(self.working_dir, output_file), "r", encoding="utf_8") as f:
            for line in f:
                if len(line) == 0 or line[len(line) - 1] != "\n":
                    # hit EOF
                    break

                event = json.loads(
                    line, object_pairs_hook=json_raise_on_duplicates)
                del event['@metadata']
                jsons.append(event)
        return jsons

    def setUp(self):
        self.template_env = jinja2.Environment(
            loader=jinja2.FileSystemLoader(self.template_paths)
        )

        # create working dir
        self.working_dir = os.path.abspath(os.path.join(
            self.build_path + "run", self.id()))
        if os.path.exists(self.working_dir):
            shutil.rmtree(self.working_dir)
        os.makedirs(self.working_dir)

        fields_yml = os.path.join(self.beat_path, "fields.yml")
        # Only add it if it exists
        if os.path.isfile(fields_yml):
            shutil.copyfile(fields_yml, os.path.join(
                self.working_dir, "fields.yml"))

        try:
            # update the last_run link
            if os.path.islink(self.build_path + "last_run"):
                os.unlink(self.build_path + "last_run")
            os.symlink(self.build_path + f"run/{self.id()}",
                       self.build_path + "last_run")
        except BaseException:
            # symlink is best effort and can fail when
            # running tests in parallel
            pass

    def wait_until(self, cond, max_timeout=10, poll_interval=0.1, name="cond"):
        """
        TODO: this can probably be a "wait_until_output_count", among other things, since that could actually use `self`, and this can become an internal function
        Waits until the cond function returns true,
        or until the max_timeout is reached. Calls the cond
        function every poll_interval seconds.

        If the max_timeout is reached before cond() returns
        true, an exception is raised.
        """
        start = datetime.now()
        while not cond():
            if datetime.now() - start > timedelta(seconds=max_timeout):
                raise WaitTimeoutError(
                    f"Timeout waiting for condition '{name}'. Waited {max_timeout} seconds.")
            time.sleep(poll_interval)

    def wait_until_output_has_key(self, key: str, max_timeout=15):
        """
        a convenience function that will wait until we see a given key in an output event
        """
        self.wait_until(
            lambda: self.output_has_key(key),
            max_timeout=max_timeout, name=f"key '{key}' to appear in output")

    def get_log(self, logfile=None):
        """
        Returns the log as a string.
        """
        if logfile is None:
            logfile = self.beat_name + "-" + self.today + ".ndjson"

        with open(os.path.join(self.working_dir, logfile), 'r', encoding="utf_8") as f:
            data = f.read()

        return data

    def get_log_lines(self, logfile=None):
        """
        Returns the log lines as a list of strings
        """
        if logfile is None:
            logfile = self.beat_name + "-" + self.today + ".ndjson"

        with open(os.path.join(self.working_dir, logfile), 'r', encoding="utf_8") as f:
            data = f.readlines()

        return data

    def wait_log_contains(self, msg, logfile=None,
                          max_timeout=10, poll_interval=0.1,
                          ignore_case=False):
        """
        wait_log_contains will wait until the log contains a given message
        """
        self.wait_until(
            cond=lambda: self.log_contains(
                msg, logfile, ignore_case=ignore_case),
            max_timeout=max_timeout,
            poll_interval=poll_interval,
            name="log_contains")

    def log_contains(self, msg, logfile=None, ignore_case=False):
        """
        Returns true if the give logfile contains the given message.
        Note that the msg must be present in a single line.
        """

        return self.log_contains_count(msg, logfile, ignore_case=ignore_case) > 0

    def log_contains_count(self, msg, logfile=None, ignore_case=False):
        """
        Returns the number of appearances of the given string in the log file
        """
        is_regexp = isinstance(msg, REGEXP_TYPE)

        counter = 0
        if ignore_case:
            msg = msg.lower()

        # Init defaults
        if logfile is None:
            logfile = self.beat_name + "-" + self.today + ".ndjson"

        print("logfile", logfile, self.working_dir)
        try:
            with open(os.path.join(self.working_dir, logfile), "r", encoding="utf_8") as f:
                for line in f:
                    if is_regexp:
                        if msg.search(line) is not None:
                            counter = counter + 1
                        continue
                    if ignore_case:
                        line = line.lower()
                    if line.find(msg) >= 0:
                        counter = counter + 1
        except IOError as ioe:
            print(ioe)
            counter = -1

        return counter

    def log_contains_countmap(self, pattern, capture_group, logfile=None):
        """
        Returns a map of the number of appearances of each captured group in the log file
        """
        counts = {}

        if logfile is None:
            logfile = self.beat_name + "-" + self.today + ".ndjson"

        try:
            with open(os.path.join(self.working_dir, logfile), "r", encoding="utf_8") as f:
                for line in f:
                    res = pattern.search(line)
                    if res is not None:
                        capt = res.group(capture_group)
                        if capt in counts:
                            counts[capt] += 1
                        else:
                            counts[capt] = 1
        except IOError:
            pass

        return counts

    def output_lines(self, output_file=None):
        """ Count number of lines in a file."""
        if output_file is None:
            output_file = self.default_output_file()

        try:
            with open(os.path.join(self.working_dir, output_file), "r", encoding="utf_8") as beat_output:
                return sum([1 for line in beat_output])
        except IOError:
            return 0

    def output_has(self, lines, output_file=None):
        """
        Returns true if the output has a given number of lines.
        """

        # Init defaults
        if output_file is None:
            output_file = self.default_output_file()
        try:
            with open(os.path.join(self.working_dir, output_file, ), "r", encoding="utf_8") as beat_output:
                return len([1 for line in beat_output]) == lines
        except IOError:
            return False

    def output_has_key(self, key: str, output_file=None):
        """
        output_has_key returns true if the given key is found in the list of events
        """

        # Awkward try/except here is for the "upstream" wait functions, if the file hasn't been created yet, it will handle the retry.
        try:
            lines = self.read_output(
                output_file=output_file, required_fields=["@timestamp"])
        except IOError:
            return False

        for line in lines:
            if key in line:
                return True
        return False

    def output_is_empty(self, output_file=None):
        """
        Returns true if the output is empty.
        """

        # Init defaults
        if output_file is None:
            output_file = self.default_output_file()

        try:
            with open(os.path.join(self.working_dir, output_file, ), "r", encoding="utf_8") as beat_file:
                return len([1 for line in beat_file]) == 0
        except IOError:
            return True

    def output_has_message(self, message, output_file=None):
        """
        Returns true if the output has the given message field.
        """
        try:
            return any(line for line in self.read_output(output_file=output_file, required_fields=["message"])
                       if line.get("message") == message)
        except (IOError, TypeError):
            return False

    def all_have_fields(self, objs, fields):
        """
        Checks that the given list of output objects have
        all the given fields.
        Raises Exception if not true.
        """
        for field in fields:
            if not all([field in o for o in objs]):
                raise Exception(f"Not all objects have a '{field}' field")

    def all_have_only_fields(self, objs, fields):
        """
        Checks if the given list of output objects have all
        and only the given fields.
        Raises Exception if not true.
        """
        self.all_have_fields(objs, fields)
        self.all_fields_are_expected(objs, fields)

    def all_fields_are_expected(self, objs, expected_fields):
        """
        Checks that all fields in the objects are from the
        given list of expected fields.
        """
        for o in objs:
            for key in o.keys():
                known = key in expected_fields
                ismeta = key.startswith('@metadata.')
                if not(known or ismeta):
                    raise Exception(f"Unexpected key '{key}' found")

    def load_fields(self, fields_doc=None):
        """
        Returns a list of fields to expect in the output dictionaries
        and a second list that contains the fields that have a
        dictionary type.

        Reads these lists from the fields documentation.
        """

        if fields_doc is None:
            fields_doc = self.beat_path + "/fields.yml"

        def extract_fields(doc_list, name):
            fields = []
            dictfields = []
            aliases = []

            if doc_list is None:
                return fields, dictfields, aliases

            for field in doc_list:

                # Skip fields without name entry
                if "name" not in field:
                    continue

                # Chain together names. Names in group `base` are top-level.
                if name != "" and name != "base":
                    newName = name + "." + field["name"]
                else:
                    newName = field["name"]

                if field.get("type") == "group":
                    subfields, subdictfields, subaliases = extract_fields(
                        field["fields"], newName)
                    fields.extend(subfields)
                    dictfields.extend(subdictfields)
                    aliases.extend(subaliases)
                else:
                    fields.append(newName)
                    if field.get("type") in ["object", "geo_point", "flattened"]:
                        dictfields.append(newName)

                if field.get("type") == "object" and field.get("object_type") == "histogram":
                    fields.append(newName + ".values")
                    fields.append(newName + ".counts")

                if field.get("type") == "alias":
                    aliases.append(newName)

            return fields, dictfields, aliases

        # TODO: Make fields_doc path more generic to work with beat-generator. If it can't find file
        # "fields.yml" you should run "make update" on metricbeat folder
        with open(fields_doc, "r", encoding="utf_8") as f:
            path = os.path.abspath(os.path.dirname(
                __file__) + "../../../../fields.yml")
            if not os.path.isfile(path):
                path = os.path.abspath(os.path.dirname(
                    __file__) + "../../../../_meta/fields.common.yml")
            with open(path, encoding="utf-8") as f2:
                content = f2.read()

            content += f.read()
            global YAML_CACHE
            content_hash = hashlib.md5(content.encode("utf-8")).hexdigest()
            doc = ""
            if content_hash in YAML_CACHE:
                doc = YAML_CACHE[content_hash]
            else:
                doc = yaml.safe_load(content)
                YAML_CACHE[content_hash] = doc

            fields = []
            dictfields = []
            aliases = []

            for item in doc:
                subfields, subdictfields, subaliases = extract_fields(
                    item["fields"], "")
                fields.extend(subfields)
                dictfields.extend(subdictfields)
                aliases.extend(subaliases)
            return fields, dictfields, aliases

    def flatten_object(self, obj, dict_fields, prefix=""):
        """
        flatten_object will flatten a beat event, turning nested keys into *.* notation
        """
        result = {}
        for key, value in obj.items():
            if isinstance(value, dict) and prefix + key not in dict_fields:
                new_prefix = prefix + key + "."
                result.update(self.flatten_object(value, dict_fields,
                                                  new_prefix))
            else:
                result[prefix + key] = value
        return result

    def copy_files(self, files, source_dir="", target_dir=""):
        """
        copy_files copies a set of files from the source to target, or the working directory if no target specified.
        """
        if not source_dir:
            source_dir = self.beat_path + "/tests/files/"
        if target_dir:
            target_dir = os.path.join(self.working_dir, target_dir)
        else:
            target_dir = self.working_dir
        for file_ in files:
            shutil.copy(os.path.join(source_dir, file_),
                        target_dir)

    def output_count(self, pred, output_file=None):
        """
        Returns true if the output line count predicate returns true
        """

        # Init defaults
        if output_file is None:
            output_file = self.default_output_file()

        try:
            with open(os.path.join(self.working_dir, output_file), "r", encoding="utf_8") as beat_out:
                return pred(len([1 for line in beat_out]))
        except IOError:
            return False

    def assert_fields_are_documented(self, evt):
        """
        Assert that all keys present in evt are documented in fields.yml.
        This reads from the global fields.yml, means `make collect` has to be run before the check.
        """
        expected_fields, dict_fields, aliases = self.load_fields()
        flat = self.flatten_object(evt, dict_fields)

        def field_pattern_match(pattern, key):
            pattern_fields = pattern.split(".")
            key_fields = key.split(".")
            if len(pattern_fields) != len(key_fields):
                return False
            for i in range(len(pattern_fields)):
                if pattern_fields[i] == "*":
                    continue
                if pattern_fields[i] != key_fields[i]:
                    return False
            return True

        def is_documented(key, docs):
            if key in docs:
                return True
            for pattern in (f for f in docs if "*" in f):
                if field_pattern_match(pattern, key):
                    return True
            return False

        for key in flat.keys():
            meta_key = key.startswith('@metadata.')
            # Range keys as used in 'date_range' etc will not have docs of course
            is_range_key = key.split('.')[-1] in ['gte', 'gt', 'lte', 'lt']
            if not(is_documented(key, expected_fields) or meta_key or is_range_key):
                raise Exception(
                    f"Key '{key}' found in event ({str(evt)}) is not documented!")
            if is_documented(key, aliases):
                raise Exception(
                    "Key '{key}' found in event is documented as an alias!")

    def get_beat_version(self):
        """
        get_beat_version returns the beats version from the exe
        """
        proc = self.start_beat(extra_args=["version"], output="version")
        proc.wait()

        return self.get_log_lines(logfile="version")[0].split()[2]

    def assert_explicit_ecs_version_set(self, module, fileset):
        """
        Assert that the module explicitly sets the ECS version field.
        """
        def get_config_paths(modules_path, module, fileset):
            fileset_path = os.path.abspath(modules_path +
                                           "/" +
                                           module +
                                           "/" +
                                           fileset +
                                           "/")
            paths = []
            for x in ["config/*.yml", "ingest/*.yml", "ingest/*.json"]:
                pathname = os.path.join(fileset_path, x)
                paths.extend(glob.glob(pathname))

            return paths

        def is_ecs_version_set(path):
            # parsing the yml file would be better but go templates in
            # the file make that difficult
            with open(path, encoding="utf-8") as fhandle:
                for line in fhandle:
                    if re.search(r"ecs\.version", line):
                        return True
            return False

        for cfg_path in get_config_paths(self.modules_path, module, fileset):  # pylint: disable=no-member
            if is_ecs_version_set(cfg_path):
                return
        raise Exception(
            f"{module}/{fileset} ecs.version not explicitly set in config or pipeline")
