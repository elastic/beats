import os
import sys
from packetbeat import BaseTest

sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), '../../../libbeat/tests/system')))

from beat import common_tests


class Test(BaseTest, common_tests.TestExportsMixin):
    pass
