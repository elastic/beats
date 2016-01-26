from topbeat import BaseTest
import numbers

"""
Contains tests for filesystem statistics.
"""


class Test(BaseTest):
    def test_filesystems(self):
        """
        Checks that system wide stats are found in the output and
        have the expected types.
        """
        self.render_config_template(
            system_stats=False,
            process_stats=False,
            filesystem_stats=True
        )
        topbeat = self.start_beat()
        self.wait_until(lambda: self.log_contains(msg="output worker: publish"))
        topbeat.check_kill_and_wait()

        output = self.read_output()[0]

        for key in [
            "fs.device_name",
            "fs.mount_point",
        ]:
            assert isinstance(output[key], basestring)

        for key in [
            "fs.used_p",
        ]:
            assert isinstance(output[key], numbers.Number)

        for key in [
            "fs.avail",
            "fs.files",
            "fs.free_files",
            "fs.total",
            "fs.used",
        ]:
            assert type(output[key]) is int or type(output[key]) is long
