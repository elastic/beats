from base import BaseTest
from beat import common_tests

import json
import os
import shutil
import signal
import subprocess
import sys
import unittest


class Test(BaseTest, common_tests.TestExportsMixin):


    def test_persistent_uuid(self):
        self.render_config_template()

        # run starts and kills the beat, reading the meta file while
        # the beat is alive
        def run():
            proc = self.start_beat(extra_args=["-path.home", self.working_dir])
            self.wait_until(lambda: self.log_contains("Mockbeat is alive"),
                            max_timeout=60)

            # open meta file before killing the beat, checking the file being
            # available right after startup
            metaFile = os.path.join(self.working_dir, "data", "meta.json")
            with open(metaFile) as f:
                meta = json.loads(f.read())

            proc.check_kill_and_wait()
            return meta

        meta0 = run()
        assert self.log_contains("Beat ID: {}".format(meta0["uuid"]))

        # remove log, restart beat and check meta file did not change
        # and same UUID is used in log output.
        os.remove(os.path.join(self.working_dir, "mockbeat-" + self.today + ".ndjson"))
        meta1 = run()
        assert self.log_contains("Beat ID: {}".format(meta1["uuid"]))

        # check meta file did not change between restarts
        assert meta0 == meta1
