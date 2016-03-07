from topbeat import BaseTest

"""
Contains tests for filtering.
"""


class Test(BaseTest):
    def test_dropfields(self):
        """
        Check drop_fields filtering action
        """
        self.render_config_template(
            system_stats=False,
            process_stats=True,
            filesystem_stats=False,
            drop_fields=["proc"]
        )
        topbeat = self.start_beat()
        self.wait_until(
            lambda: self.log_contains(
                "output worker: publish"))
        topbeat.kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp", "type"],
        )[0]

        for key in [
            "proc.cpu.start_time",
            "proc.cpu.total",
            "proc.cpu.total_p",
            "proc.cpu.user",
            "proc.cpu.system",
            "proc.name",
            "proc.state",
            "proc.pid",
        ]:
            assert key not in output

    def test_include_fields(self):
        """
        Check include_fields filtering action
        """
        self.render_config_template(
            system_stats=False,
            process_stats=True,
            filesystem_stats=False,
            include_fields=["proc.cpu", "proc.mem"]
        )
        topbeat = self.start_beat()
        self.wait_until(
            lambda: self.log_contains(
                "output worker: publish"))

        topbeat.kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp", "type"],
        )[0]
        print(output)

        for key in [
            "proc.cpu.start_time",
            "proc.cpu.total",
            "proc.cpu.total_p",
            "proc.cpu.user",
            "proc.cpu.system",
            "proc.mem.size",
            "proc.mem.rss",
            "proc.mem.rss_p"
        ]:
            assert key in output

        for key in [
            "proc.name",
            "proc.pid",
        ]:
            assert key not in output

    def test_multiple_actions(self):
        """
        Check the result when configuring two actions: include_fields
        and drop_fields.
        """
        self.render_config_template(
            system_stats=False,
            process_stats=True,
            filesystem_stats=False,
            include_fields=["proc"],
            drop_fields=["proc.mem"],
        )
        topbeat = self.start_beat()
        self.wait_until(
            lambda: self.log_contains(
                "output worker: publish"))

        topbeat.kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp", "type"],
        )[0]

        for key in [
            "proc.cpu.start_time",
            "proc.cpu.total",
            "proc.cpu.total_p",
            "proc.cpu.user",
            "proc.cpu.system",
            "proc.name",
            "proc.pid",
        ]:
            assert key in output

        for key in [
            "proc.mem.size",
            "proc.mem.rss",
            "proc.mem.rss_p"
        ]:
            assert key not in output

    def test_contradictory_multiple_actions(self):
        """
        Check the behaviour of a contradictory multiple actions
        """
        self.render_config_template(
            system_stats=False,
            process_stats=True,
            filesystem_stats=False,
            include_fields=["proc.mem.size", "proc.mem.rss_p"],
            drop_fields=["proc.mem.size", "proc.mem.rss_p"],
        )
        topbeat = self.start_beat()
        self.wait_until(
            lambda: self.log_contains(
                "output worker: publish"))

        topbeat.kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp", "type"],
        )[0]

        for key in [
            "proc.mem.size",
            "proc.mem.rss",
            "proc.cpu.start_time",
            "proc.cpu.total",
            "proc.cpu.total_p",
            "proc.cpu.user",
            "proc.cpu.system",
            "proc.name",
            "proc.pid",
            "proc.mem.rss_p"
        ]:
            assert key not in output
