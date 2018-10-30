import sys
import os


sys.path.append(os.path.join(os.path.dirname(__file__),
                             '../../../../libbeat/tests/system'))


from base import BaseTest


INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class TestManagement(BaseTest):

    def test_broken(self):
        assert False
