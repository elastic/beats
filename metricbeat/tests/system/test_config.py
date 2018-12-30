import os
import metricbeat
import unittest
from nose.plugins.attrib import attr
import urllib2
import time


class ConfigTest(metricbeat.BaseTest):
    def get_host(self):
        return 'http://' + os.getenv('ES_HOST', 'localhost') + ':' + os.getenv('ES_PORT', '9200')
