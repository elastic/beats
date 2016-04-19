import subprocess
import jinja2
import unittest
import os
import shutil
import json
import time
from datetime import datetime, timedelta

build_path = "../../build/system-tests/"

class Proc(object):
    """
    Slim wrapper on subprocess.Popen that redirects
    both stdout and stderr to a file on disk and makes
    sure to stop the process and close the output file when
    the object gets collected.
    """
    def __init__(self, args, outputfile):
        self.args = args
        self.output = open(outputfile, "wb")

    def start(self):

        self.stdin_read, self.stdin_write = os.pipe()

        self.proc = subprocess.Popen(
            self.args,
            stdin=self.stdin_read,
            stdout=self.output,
            stderr=subprocess.STDOUT,
            bufsize=0,
        )
        return self.proc

    def wait(self):
        return self.proc.wait()

    def kill_and_wait(self):
        self.proc.terminate()
        os.close(self.stdin_write)
        return self.proc.wait()

    def __del__(self):
        try:
            self.output.close()
        except:
            pass
        try:
            self.proc.terminate()
            self.proc.kill()
        except:
            pass


class TestCase(unittest.TestCase):

    def run_filebeat(self, cmd="../../filebeat.test",
                     config="filebeat.yml",
                     output="filebeat.log",
                     extra_args=[],
                     debug_selectors=[]):
        """
        Executes filebeat
        Waits for the process to finish before returning to
        the caller.
        """

        args = [cmd]

        args.extend(["-e",
                     "-c", os.path.join(self.working_dir, config),
                     "-t",
                     "-systemTest",
                     "-v",
                     "-d", "*",
                     "-test.coverprofile",
                     os.path.join(self.working_dir, "coverage.cov")
                     ])
        if extra_args:
            args.extend(extra_args)

        if debug_selectors:
            args.extend(["-d", ",".join(debug_selectors)])

        with open(os.path.join(self.working_dir, output), "wb") as outputfile:
            proc = subprocess.Popen(args,
                                    stdout=outputfile,
                                    stderr=subprocess.STDOUT)
            return proc.wait()

    def start_filebeat(self,
                       cmd="../../filebeat.test",
                       config="filebeat.yml",
                       output="filebeat.log",
                       extra_args=[],
                       debug_selectors=[]):
        """
        Starts filebeat and returns the process handle. The
        caller is responsible for stopping / waiting for the
        Proc instance.
        """
        args = [cmd,
                "-e",
                "-c", os.path.join(self.working_dir, config),
                "-systemTest",
                "-v",
                "-d", "*",
                "-test.coverprofile",
                os.path.join(self.working_dir, "coverage.cov")
                ]
        if extra_args:
            args.extend(extra_args)

        if debug_selectors:
            args.extend(["-d", ",".join(debug_selectors)])

        proc = Proc(args, os.path.join(self.working_dir, output))
        proc.start()
        return proc

    def render_config_template(self, template="filebeat.yml.j2",
                               output="filebeat.yml", **kargs):
        template = self.template_env.get_template(template)
        kargs["fb"] = self
        output_str = template.render(**kargs)
        with open(os.path.join(self.working_dir, output), "wb") as f:
            f.write(output_str)

    def read_output(self, output_file="output/filebeat"):
        jsons = []
        with open(os.path.join(self.working_dir, output_file), "r") as f:
            for line in f:
                jsons.append(self.flatten_object(json.loads(line),
                                                 []))
        self.all_have_fields(jsons, ["@timestamp", "type",
                                     "beat.name", "beat.hostname", "count"])
        return jsons

    def copy_files(self, files, source_dir="files/", target_dir=""):
        if target_dir:
            target_dir = os.path.join(self.working_dir, target_dir)
        else:
            target_dir = self.working_dir
        for file_ in files:
            shutil.copy(os.path.join(source_dir, file_),
                        target_dir)

    def setUp(self):

        self.template_env = jinja2.Environment(
            loader=jinja2.FileSystemLoader("config")
        )

        # create working dir
        self.working_dir = os.path.join(build_path + "run", self.id())
        if os.path.exists(self.working_dir):
            shutil.rmtree(self.working_dir)
        os.makedirs(self.working_dir)

        try:
            # update the last_run link
            if os.path.islink(build_path + "last_run"):
                os.unlink(build_path + "last_run")
            os.symlink(build_path + "run/{}".format(self.id()), build_path + "last_run")
        except:
            # symlink is best effort and can fail when
            # running tests in parallel
            pass

    def wait_until(self, cond, max_timeout=10, poll_interval=0.1, name="cond"):
        """
        Waits until the cond function returns true,
        or until the max_timeout is reached. Calls the cond
        function every poll_interval seconds.

        If the max_timeout is reached before cond() returns
        true, an exception is raised.
        """
        start = datetime.now()
        while not cond():
            if datetime.now() - start > timedelta(seconds=max_timeout):
                raise Exception("Timeout waiting for '{}' to be true. "
                                .format(name) +
                                "Waited {} seconds.".format(max_timeout))
            time.sleep(poll_interval)

    def log_contains(self, msg, logfile="filebeat.log"):
        """
        Returns true if the give logfile contains the given message.
        Note that the msg must be present in a single line.
        """
        try:
            with open(os.path.join(self.working_dir, logfile), "r") as f:
                for line in f:
                    if line.find(msg) >= 0:
                        return True
                return False
        except IOError:
            return False

    def output_has(self, lines, output_file="output/filebeat"):
        """
        Returns true if the output has a given number of lines.
        """
        return self.output_count(
            lambda x: lines == x,
            output_file)

    def output_between(self, start, end, output_file="output/filebeat"):
        return self.output_count(
            lambda x: start <= x <= end,
            output_file)

    def output_count(self, pred, output_file="output/filebeat"):
        """
        Returns true if the output line count predicate returns true
        """
        try:
            with open(os.path.join(self.working_dir, output_file), "r") as f:
                return pred(len([1 for line in f]))
        except IOError:
            return False

    def all_have_fields(self, objs, fields):
        """
        Checks that the given list of output objects have
        all the given fields.
        Raises Exception if not true.
        """
        for field in fields:
            if not all([field in o for o in objs]):
                raise Exception("Not all objects have a '{}' field"
                                .format(field))

    def all_have_only_fields(self, objs, fields):
        """
        Checks if the given list of output objects have all
        and only the given fields.
        Raises Exception if not true.
        """
        self.all_have_fields(objs, fields)
        self.all_fields_are_expected(objs, fields)

    def all_fields_are_expected(self, objs, expected_fields,
                                dict_fields=[]):
        """
        Checks that all fields in the objects are from the
        given list of expected fields.
        """
        for o in objs:
            for key in o.keys():
                if key not in dict_fields and key not in expected_fields:
                    raise Exception("Unexpected key '{}' found"
                                    .format(key))

    def flatten_object(self, obj, dict_fields, prefix=""):
        result = {}
        for key, value in obj.items():
            if isinstance(value, dict) and prefix + key not in dict_fields:
                new_prefix = prefix + key + "."
                result.update(self.flatten_object(value, dict_fields,
                                                  new_prefix))
            else:
                result[prefix + key] = value
        return result

    def get_dot_filebeat(self):
        # Returns content of the .filebeat file
        dotFilebeat = self.working_dir + '/.filebeat'
        assert os.path.isfile(dotFilebeat) is True

        with open(dotFilebeat) as file:
            return json.load(file)


    def log_contains_count(self, msg, logfile=None):
        """
        Returns the number of appearances of the given string in the log file
        """

        counter = 0

        # Init defaults
        if logfile is None:
            logfile = "filebeat.log"

        try:
            with open(os.path.join(self.working_dir, logfile), "r") as f:
                for line in f:
                    if line.find(msg) >= 0:
                        counter = counter + 1
        except IOError:
            counter = -1

        return counter
