import os
import sys
from beat import common_tests
from packetbeat import BaseTest


class Test(BaseTest, common_tests.TestExportsMixin, common_tests.TestDashboardMixin):
    pass
