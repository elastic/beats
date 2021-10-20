import os
import sys
import unittest
import pytest
import semver
import requests
import shutil
from beat import common_tests
from beat.beat import INTEGRATION_TESTS
from elasticsearch import Elasticsearch
from packetbeat import BaseTest


class Test(BaseTest, common_tests.TestExportsMixin, common_tests.TestDashboardMixin):
    pass

