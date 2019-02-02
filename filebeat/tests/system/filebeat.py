import json
import os
import stat
import sys

sys.path.append(os.path.join(os.path.dirname(__file__), '../../../libbeat/tests/system'))

from beat.beat import TestCase

default_registry_file = 'registry/filebeat/data.json'


class BaseTest(TestCase):

    @classmethod
    def setUpClass(self):
        if not hasattr(self, "beat_name"):
            self.beat_name = "filebeat"
        if not hasattr(self, "beat_path"):
            self.beat_path = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../"))

        super(BaseTest, self).setUpClass()

    def has_registry(self, name=None, data_path=None):
        if not name:
            name = default_registry_file
        if not data_path:
            data_path = self.working_dir

        dotFilebeat = os.path.join(data_path, name)
        return os.path.isfile(dotFilebeat)

    def get_registry(self, name=None, data_path=None):
        if not name:
            name = default_registry_file
        if not data_path:
            data_path = self.working_dir

        # Returns content of the registry file
        dotFilebeat = os.path.join(data_path, name)
        self.wait_until(cond=lambda: os.path.isfile(dotFilebeat))

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

    def file_permissions(self, path):
        full_path = os.path.join(self.working_dir, path)
        return oct(stat.S_IMODE(os.lstat(full_path).st_mode))
