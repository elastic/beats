import os
import shutil
import metricbeat


class TestCommands(metricbeat.BaseTest):
    """
    Test metricbeat subcommands
    """

    def setUp(self):
        super(TestCommands, self).setUp()

        # Enable modules reload with default paths
        self.render_config_template(reload=True)
        os.mkdir(self.working_dir + "/modules.d")

    def test_modules_list(self):
        """
        Test modules list command
        """
        self.touch(self.working_dir + "/modules.d/enabled.yml")
        self.touch(self.working_dir + "/modules.d/disabled.yml.disabled")

        exit_code = self.run_beat(logging_args=None,
                                  extra_args=["modules", "list"])

        assert exit_code == 0
        assert "Enabled:\nenabled" in self.get_log()
        assert "Disabled:\ndisabled" in self.get_log()

        # Add one more disabled module
        self.touch(self.working_dir + "/modules.d/disabled2.yml.disabled")
        exit_code = self.run_beat(logging_args=None,
                                  extra_args=["modules", "list"])

        assert exit_code == 0
        assert "Enabled:\nenabled" in self.get_log()
        assert "Disabled:\ndisabled\ndisabled2" in self.get_log()

    def test_modules_enable(self):
        """
        Test modules enable command
        """
        self.touch(self.working_dir + "/modules.d/enabled.yml")
        self.touch(self.working_dir + "/modules.d/disabled1.yml.disabled")
        self.touch(self.working_dir + "/modules.d/disabled2.yml.disabled")
        self.touch(self.working_dir + "/modules.d/disabled3.yml.disabled")

        # Enable one module
        exit_code = self.run_beat(
            extra_args=["modules", "enable", "disabled1"])
        assert exit_code == 0

        assert self.log_contains("Enabled disabled1")
        assert os.path.exists(self.working_dir + "/modules.d/disabled1.yml")
        assert not os.path.exists(
            self.working_dir + "/modules.d/disabled1.yml.disabled")
        assert os.path.exists(
            self.working_dir + "/modules.d/disabled2.yml.disabled")
        assert os.path.exists(
            self.working_dir + "/modules.d/disabled3.yml.disabled")

        # Enable several modules at once:
        exit_code = self.run_beat(
            extra_args=["modules", "enable", "disabled2", "disabled3"])
        assert exit_code == 0

        assert self.log_contains("Enabled disabled2")
        assert self.log_contains("Enabled disabled3")
        assert os.path.exists(self.working_dir + "/modules.d/disabled2.yml")
        assert os.path.exists(self.working_dir + "/modules.d/disabled3.yml")
        assert not os.path.exists(
            self.working_dir + "/modules.d/disabled2.yml.disabled")
        assert not os.path.exists(
            self.working_dir + "/modules.d/disabled3.yml.disabled")

    def test_modules_disable(self):
        """
        Test modules disable command
        """
        self.touch(self.working_dir + "/modules.d/enabled1.yml")
        self.touch(self.working_dir + "/modules.d/enabled2.yml")
        self.touch(self.working_dir + "/modules.d/enabled3.yml")
        self.touch(self.working_dir + "/modules.d/disabled2.yml.disabled")

        # Disable one module
        exit_code = self.run_beat(
            extra_args=["modules", "disable", "enabled1"])
        assert exit_code == 0

        assert self.log_contains("Disabled enabled1")
        assert os.path.exists(
            self.working_dir + "/modules.d/enabled1.yml.disabled")
        assert not os.path.exists(self.working_dir + "/modules.d/enabled1.yml")
        assert os.path.exists(self.working_dir + "/modules.d/enabled2.yml")
        assert os.path.exists(self.working_dir + "/modules.d/enabled3.yml")

        # Disable several modules at once:
        exit_code = self.run_beat(
            extra_args=["modules", "disable", "enabled2", "enabled3"])
        assert exit_code == 0

        assert self.log_contains("Disabled enabled2")
        assert self.log_contains("Disabled enabled3")
        assert os.path.exists(
            self.working_dir + "/modules.d/enabled2.yml.disabled")
        assert os.path.exists(
            self.working_dir + "/modules.d/enabled3.yml.disabled")
        assert not os.path.exists(self.working_dir + "/modules.d/enabled2.yml")
        assert not os.path.exists(self.working_dir + "/modules.d/enabled3.yml")

    def test_modules_test(self):
        """
        Test test modules command
        """
        self.write_system_yml()

        exit_code = self.run_beat(
            logging_args=None,
            extra_args=["test", "modules"])

        assert exit_code == 0
        assert self.log_contains("cpu...OK")
        assert self.log_contains("memory...OK")

    def test_modules_test_error(self):
        """
        Test test modules command with an error result
        """
        self.write_system_yml()
        self.write_nginx_yml()

        exit_code = self.run_beat(
            logging_args=None,
            extra_args=["test", "modules"])

        assert exit_code == 0
        try:
            assert any((
                self.log_contains("ERROR error making http request"),
                self.log_contains("ERROR timeout waiting for an event"),
            ))
        except:
            # Print log to help debugging this if error message changes
            print self.get_log()
            raise
        assert self.log_contains("cpu...OK")
        assert self.log_contains("memory...OK")

    def test_modules_test_filter_no_result(self):
        """
        Test test modules command filter by module (no result)
        """
        self.write_system_yml()

        exit_code = self.run_beat(
            logging_args=None,
            extra_args=["test", "modules", "apache"])

        assert exit_code == 0
        assert not self.log_contains("OK")

    def test_modules_test_filter(self):
        """
        Test test modules command filter by metricset
        """
        self.write_system_yml()
        self.write_nginx_yml()

        exit_code = self.run_beat(
            logging_args=None,
            extra_args=["test", "modules", "system", "cpu"])

        assert exit_code == 0
        assert self.log_contains("cpu...OK")
        assert not self.log_contains("memory...OK")

    def touch(self, path):
        open(path, 'a').close()

    def write_system_yml(self):
        with open(self.working_dir + "/modules.d/system.yml", "wb") as f:
            f.write("""
- module: system
  period: 10s
  metricsets:
    - cpu
    - memory""")

    def write_nginx_yml(self):
        with open(self.working_dir + "/modules.d/nginx.yml", "wb") as f:
            f.write("""
- module: nginx
  period: 10s
  hosts: ["errorhost:80"]
  metricsets:
    - stubstatus""")
