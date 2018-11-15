import os
import metricbeat
import unittest
from nose.plugins.attrib import attr
import urllib2
import time


class ConfigTest(metricbeat.BaseTest):
    def get_host(self):
        return 'http://' + self.compose_host()
