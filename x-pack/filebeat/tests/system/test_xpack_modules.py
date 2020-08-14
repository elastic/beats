import os
import sys

sys.path.append(os.path.join(os.path.dirname(__file__), '../../../../filebeat/tests/system'))

import test_modules


class XPackTest(test_modules.Test):

    @classmethod
    def setUpClass(self):
        self.beat_name = "filebeat"
        self.beat_path = os.path.abspath(
            os.path.join(os.path.dirname(__file__), "../../"))

        super(test_modules.Test, self).setUpClass()

    def setUp(self):
        super(test_modules.Test, self).setUp()
