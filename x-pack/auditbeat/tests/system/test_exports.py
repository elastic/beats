import jinja2
import os
import platform
import sys
import time
import unittest

from auditbeat_xpack import *
from beat import common_tests


class Test(AuditbeatXPackTest, common_tests.TestExportsMixin):
    pass
