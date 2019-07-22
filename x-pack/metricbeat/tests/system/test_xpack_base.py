import os

import xpack_metricbeat
import test_base


class Test(xpack_metricbeat.XPackTest, test_base.Test):
    def kibana_dir(self):
        return os.path.join(self.beat_path, 'build', 'kibana')
