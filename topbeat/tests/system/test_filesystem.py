from topbeat import TestCase


"""
Contains tests for ide statistics.
"""


class Test(TestCase):
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
        topbeat = self.start_topbeat()
        self.wait_until(lambda: self.output_has(lines=1))
        topbeat.kill_and_wait()

        output = self.read_output()[0]

        for key in [
            "fs.device_name",
            "fs.mount_point",
        ]:
            assert isinstance(output[key], basestring)

        for key in [
            "fs.used_p",
        ]:
            assert type(output[key]) is float

        for key in [
            "fs.avail",
            "fs.files",
            "fs.free_files",
            "fs.total",
            "fs.used",
        ]:
            assert type(output[key]) is int or type(output[key]) is long
