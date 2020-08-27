import os
import sys
from packetbeat import BaseTest


from beat import common_tests


class Test(BaseTest, common_tests.TestExportsMixin):
    pass
