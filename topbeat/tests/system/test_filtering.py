from topbeat import BaseTest
import numbers

"""
Contains tests for filtering.
"""


class Test(BaseTest):
    def test_dropfields(self):
        """
        Check filtering by applying a drop fields action.
        """
        self.render_config_template(
            system_stats=False,
            process_stats=True,
            filesystem_stats=False,
            drop_fields=["proc.mem"]
        )
        topbeat = self.start_beat()
        self.wait_until(
            lambda: self.log_contains(
                "output worker: publish"))
        topbeat.kill_and_wait()

        output = self.read_output()[0]

        for key in [
            "proc.cpu.start_time",
            "proc.cpu.total",
            "proc.cpu.total_p",
            "proc.cpu.user",
            "proc.cpu.system",
            "proc.name",
            "proc.state",
            "proc.pid",
            "beat.hostname",
            "type"
        ]:
            assert key in output

        for key in [
            "proc.mem.size",
            "proc.mem.rss",
            "proc.mem.rss_p"
        ]:
            assert key not in output

    def test_includefields(self):
        """
        Check filtering by applying an include fields action
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

        output = self.read_output()[0]

        for key in [
            "proc.cpu.start_time",
            "proc.cpu.total",
            "proc.cpu.total_p",
            "proc.cpu.user",
            "proc.cpu.system",
            "beat.hostname",
            "type",
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

    def test_include_and_drop_fields(self):
        """
        Check filtering by applying an include fields action
        followed by drop fields.
        """
        self.render_config_template(
            system_stats=False,
            process_stats=True,
            filesystem_stats=True,
            drop_fields=["fs"],
            include_fields=["proc.cpu", "proc.mem"]
        )
        topbeat = self.start_beat()
        self.wait_until(
            lambda: self.log_contains(
                "output worker: publish"))

        topbeat.kill_and_wait()

        for output in self.read_output():
            assert output["type"] == "process"

            for key in [
                "proc.cpu.start_time",
                "proc.cpu.total",
                "proc.cpu.total_p",
                "proc.cpu.user",
                "proc.cpu.system",
                "beat.hostname",
                "type",
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
