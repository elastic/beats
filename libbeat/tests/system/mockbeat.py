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
        self.proc = subprocess.Popen(
            self.args,
            stdout=self.output,
            stderr=subprocess.STDOUT)
        return self.proc

    def wait(self):
        return self.proc.wait()

    def kill_and_wait(self):
        self.proc.terminate()
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

    def run_mockbeat(self, cmd="../../libbeat.test",
                     config="mockbeat.yml",
                     output="mockbeat.log",
                     extra_args=[],
                     debug_selectors=[]):
        """
        Executes mockbeat
        Waits for the process to finish before returning to
        the caller.
        """

        args = [cmd]

        args.extend(["-e",
                     "-c", os.path.join(self.working_dir, config),
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

    def start_mockbeat(self,
                       cmd="../../libbeat.test",
                       config="mockbeat.yml",
                       output="mockbeat.log",
                       extra_args=[],
                       debug_selectors=[]):
        """
        Starts mockbeat and returns the process handle. The
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

    def render_config_template(self, template="mockbeat.yml.j2",
                               output="mockbeat.yml", **kargs):
        template = self.template_env.get_template(template)
        kargs["fb"] = self
        output_str = template.render(**kargs)
        with open(os.path.join(self.working_dir, output), "wb") as f:
            f.write(output_str)

    def read_output(self, output_file="output/mockbeat"):
        jsons = []
        with open(os.path.join(self.working_dir, output_file), "r") as f:
            for line in f:
                jsons.append(self.flatten_object(json.loads(line),
                                                 []))
        self.all_have_fields(jsons, ["@timestamp", "type",
                                     "shipper", "count"])
        return jsons

    def copy_files(self, files, source_dir="files/"):
        for file_ in files:
            shutil.copy(os.path.join(source_dir, file_),
                        self.working_dir)

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

    def log_contains(self, msg, logfile="mockbeat.log"):
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

    def output_has(self, lines, output_file="output/mockbeat"):
        """
        Returns true if the output has a given number of lines.
        """
        try:
            with open(os.path.join(self.working_dir, output_file), "r") as f:
                return len([1 for line in f]) == lines
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
