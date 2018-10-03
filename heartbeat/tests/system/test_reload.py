from heartbeat import BaseTest
import nose.tools
import os


class Test(BaseTest):
    def __init__(self, *args):
        self.proc = None
        super(Test, self).__init__(*args)

    def test_config_reload(self):
        """
        Test a reload of a config
        """
        server = self.start_server("hello world", 200)
        try:
            self.setup()

            cfg_file = "test.yml"

            self.write_dyn_config(
                cfg_file, self.http_cfg("http://localhost:8185"))

            self.wait_until(lambda: self.output_has(lines=1))

            self.assert_last_status("up")

            self.write_dyn_config(
                cfg_file, self.http_cfg("http://localhost:8186"))

            self.wait_until(lambda: self.last_output_line()[
                            "http.url"] == "http://localhost:8186")

            self.assert_last_status("down")

            self.proc.check_kill_and_wait()
        finally:
            server.shutdown()

    def test_config_remove(self):
        """
        Test the removal of a dynamic config
        """
        server = self.start_server("hello world", 200)
        try:
            self.setup()

            cfg_file = "test.yml"

            self.write_dyn_config(
                cfg_file, self.http_cfg("http://localhost:8185"))

            self.wait_until(lambda: self.output_has(lines=2))

            self.assert_last_status("up")

            os.remove(self.monitors_dir() + cfg_file)

            # Ensure the job was removed from the scheduler
            self.wait_until(lambda: self.log_contains(
                "Remove scheduler job 'http@http://localhost:8185"))
            self.wait_until(lambda: self.log_contains(
                "Job 'http@http://localhost:8185' returned"))

            self.proc.check_kill_and_wait()
        finally:
            server.shutdown()

    def test_config_add(self):
        """
        Test the addition of a dynamic config
        """
        self.setup()

        self.wait_until(lambda: self.log_contains(
            "Starting reload procedure, current runners: 0"))

        server = self.start_server("hello world", 200)
        try:
            self.write_dyn_config(
                "test.yml", self.http_cfg("http://localhost:8185"))

            self.wait_until(lambda: self.log_contains(
                "Starting reload procedure, current runners: 1"))

            self.wait_until(lambda: self.output_has(lines=1))

            self.proc.check_kill_and_wait()
        finally:
            server.shutdown()

    def setup(self):
        os.mkdir(self.monitors_dir())
        self.render_config_template(
            reload=True,
            reload_path=self.monitors_dir() + "*.yml",
            flush_min_events=1,
        )

        self.proc = self.start_beat()

    def write_dyn_config(self, filename, cfg):
        with open(self.monitors_dir() + filename, 'w') as f:
            f.write(cfg)

    def monitors_dir(self):
        return self.working_dir + "/monitors.d"

    def assert_last_status(self, status):
        nose.tools.eq_(self.last_output_line()["monitor.status"], status)

    def last_output_line(self):
        return self.read_output()[-1]

    @staticmethod
    def http_cfg(url):
        return """
- type: http
  schedule: "@every 1s"
  urls: ["{url}"]
        """[1:-1].format(url=url)
