import json
import os
import sys

sys.path.append('../../../libbeat/tests/system')

from beat.beat import TestCase


class BaseTest(TestCase):

    @classmethod
    def setUpClass(self):
        self.beat_name = "filebeat"
        super(BaseTest, self).setUpClass()

    def get_registry(self):
        # Returns content of the registry file
        dotFilebeat = self.working_dir + '/registry'
        assert os.path.isfile(dotFilebeat) is True

        with open(dotFilebeat) as file:
            return json.load(file)

    def get_registry_entry_by_path(self, path):
        """
        Fetches the registry file and checks if an entry for the given path exists
        If the path exists, the state for the given path is returned
        If a path exists multiple times (which is possible because of file rotation)
        the most recent version is returned
        """
        registry = self.get_registry()

        tmp_entry = None

        # Checks all entries and returns the most recent one
        for entry in registry:
            if entry["source"] == path:
                if tmp_entry == None:
                    tmp_entry = entry
                else:
                    if tmp_entry["timestamp"] < entry["timestamp"]:
                        tmp_entry = entry

        return tmp_entry

