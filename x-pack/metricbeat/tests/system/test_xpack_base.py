import xpack_metricbeat
import test_base
from beat import common_tests


class Test(xpack_metricbeat.XPackTest, test_base.Test, common_tests.TestExportsMixin):
    pass
