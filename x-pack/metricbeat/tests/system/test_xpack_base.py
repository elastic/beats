import os
import unittest

import xpack_metricbeat
import test_base
from beat import common_tests


@unittest.skip("https://github.com/elastic/beats/issues/26536")
class Test(xpack_metricbeat.XPackTest, test_base.Test, common_tests.TestExportsMixin):
    pass
