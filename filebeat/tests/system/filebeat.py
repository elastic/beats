import json
import os
import sys

sys.path.append(os.path.join(os.path.dirname(__file__), '../../../libbeat/tests/system'))

from beat.beat import TestCase


class BaseTest(TestCase):

    @classmethod
    def setUpClass(self):
        self.beat_name = "filebeat"
        self.beat_path = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../"))

        super(BaseTest, self).setUpClass()

    def get_registry(self):
        # Returns content of the registry file
        dotFilebeat = self.working_dir + '/registry'
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

    def assert_fields_are_documented(self, evt):
        """
        Assert that all keys present in evt are documented in fields.yml.
        This reads from the global fields.yml, means `make collect` has to be run before the check.
        """
        expected_fields, dict_fields = self.load_fields()
        flat = self.flatten_object(evt, dict_fields)

        for key in flat.keys():
            if key not in expected_fields:
                raise Exception("Key '{}' found in event is not documented!".format(key))
