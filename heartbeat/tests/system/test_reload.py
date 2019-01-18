from heartbeat import BaseTest
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
            self.setup_dynamic()

            cfg_file = "test.yml"

            self.write_dyn_config(
                cfg_file, self.http_cfg("myid", "http://localhost:{}".format(server.server_port)))

            self.wait_until(lambda: self.output_has(lines=1))

            self.assert_last_status("up")

            self.write_dyn_config(
                cfg_file, self.http_cfg("myid", "http://203.0.113.1:8186"))

            self.wait_until(lambda: self.last_output_line()[
                            "url.full"] == "http://203.0.113.1:8186")

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
            self.setup_dynamic()

            cfg_file = "test.yml"

            self.write_dyn_config(
                cfg_file, self.http_cfg("myid", "http://localhost:{}".format(server.server_port)))

            self.wait_until(lambda: self.output_has(lines=2))

            self.assert_last_status("up")

            os.remove(self.monitors_dir() + cfg_file)

            # Ensure the job was removed from the schduler
            self.wait_until(lambda: self.log_contains("Remove scheduler job 'myid'"))
            self.wait_until(lambda: self.log_contains("Job 'myid' returned"))

            self.proc.check_kill_and_wait()
        finally:
            server.shutdown()

    def test_config_add(self):
        """
        Test the addition of a dynamic config
        """
        self.setup_dynamic()

        self.wait_until(lambda: self.log_contains(
            "Starting reload procedure, current runners: 0"))

        server = self.start_server("hello world", 200)
        try:
            self.write_dyn_config(
                "test.yml", self.http_cfg("myid", "http://localhost:{}".format(server.server_port)))

            self.wait_until(lambda: self.log_contains(
                "Starting reload procedure, current runners: 1"))

            self.wait_until(lambda: self.output_has(lines=1))

            self.proc.check_kill_and_wait()
        finally:
            server.shutdown()
