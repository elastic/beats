from functionbeat import BaseTest

import os
import sys
import time
import boto3
import unittest
import uuid
from random import randint
from elasticsearch import Elasticsearch
from nose.tools import assert_equals

sys.path.append(os.path.join(os.path.dirname(__file__), '../../../../libbeat/tests/system'))
from beat.beat import TestCase


# Timeout on the lambda execution tests
TIMEOUT = 5 * 60
FUNCTIONBEAT_INTEGRATION_TESTS = os.environ.get('FUNCTIONBEAT_INTEGRATION_TESTS', False)


class Test(BaseTest):
    def setUp(self):
        super(Test, self).setUp()
        self.es = Elasticsearch([self.cloud_to_elasticsearch_py()], verify_certs=True)

        # cleanup anything on that cluster for functionbeat.
        self.es.indices.delete("functionbeat*")

        r = str(randint(0, 100))
        t = str(int(time.time()))

        # Random resources name
        self.log_group_name = "myloggroup-%s-%s" % (t, r)
        self.bucket = "test-bucket-%s-%s" % (t, r)
        self.log_stream_name = "mystream-" + r
        self.function_name = "functionname" + r

        # Setup the AWS environment
        self.client = boto3.client('logs')
        self.client.create_log_group(logGroupName=self.log_group_name)
        self.client.create_log_stream(logGroupName=self.log_group_name,
                                      logStreamName=self.log_stream_name)

    def tearDown(self):
        super(Test, self).tearDown()

        # cleanup anything on that cluster for functionbeat.
        self.es.indices.delete("functionbeat*")

        # Cleanup the environment
        self.client.delete_log_group(logGroupName=self.log_group_name)
        s3 = boto3.resource('s3')
        bucket = s3.Bucket(self.bucket)
        for key in bucket.objects.all():
            key.delete()
        bucket.delete()

    @unittest.skipIf(not FUNCTIONBEAT_INTEGRATION_TESTS,
                     "functionbeat integration tests are disabled, run with FUNCTIONBEAT_INTEGRATION_TESTS=1 to enable them.")
    def test_deploy_cloudwatch_local_options(self):
        """
        Deploy a function to retrieve cloudwatch logs.
        """

        # Setup the function
        self.deploy(
            deploy_bucket=self.bucket,
            function={
                'name': self.function_name,
                'enabled': True,
                'type': 'cloudwatch_logs',
                'triggers': [
                    {'log_group_name': self.log_group_name},
                ],
                'processors': [
                    {'dissect': {'tokenizer': 'id=%{id} m=%{message}'}},
                ],
                'fields': {
                    'hello': 'world',
                }
            },
            cloud_id=self.cloud_id(),
            cloud_auth=self.cloud_auth(),
        )

        uid = uuid.uuid4()
        # Insert some events into cloudwatch logs
        self.client.put_log_events(logGroupName=self.log_group_name,
                                   logStreamName=self.log_stream_name,
                                   logEvents=[
                                       {
                                           'timestamp': time_now_millis(),
                                           'message': 'id=%s m=hello world 1' % uid
                                       },
                                       {
                                           'timestamp': time_now_millis(),
                                           'message': 'id=%s m=hello world 2' % uid
                                       },
                                       {
                                           'timestamp': time_now_millis(),
                                           'message': 'id=%s m=hello world 3' % uid
                                       },
                                       {
                                           'timestamp': time_now_millis(),
                                           'message': 'id=%s m=hello world 4' % uid
                                       },
                                       {
                                           'timestamp': time_now_millis(),
                                           'message': 'id=%s m=hello world 5' % uid
                                       },
                                   ])

        self.wait_until(lambda: self.es.search(index="functionbeat*", body={"query": {"match_all": {}}})['hits']['total'] >= 5,
                        max_timeout=TIMEOUT)
        results = self.es.search(index="functionbeat*")
        messages = self.collect_messages(results)

        assert_equals(len(messages), 5)
        for i in range(0, 4):
            self.assertDictContainsSubset({
                'dissect': {
                    'id': str(uid),
                    'message': 'hello world %s' % (i+1),
                },
                'message': 'id=%s m=hello world %s' % (uid, (i+1)),
                'log_group': self.log_group_name,
                'fields': {
                    'hello': 'world',
                },
            },
                messages[i])

        self.remove()

    @unittest.skipIf(not FUNCTIONBEAT_INTEGRATION_TESTS,
                     "functionbeat integration tests are disabled, run with FUNCTIONBEAT_INTEGRATION_TESTS=1 to enable them.")
    def test_deploy_cloudwatch_global_options(self):
        """
        Deploy a function to retrieve cloudwatch logs with global options.
        """

        self.deploy(deploy_bucket=self.bucket,
                    function={
                        'name': self.function_name,
                        'enabled': True,
                        'type': 'cloudwatch_logs',
                        'triggers': [
                            {'log_group_name': self.log_group_name},
                        ],
                    },
                    cloud_id=self.cloud_id(),
                    cloud_auth=self.cloud_auth(),
                    options={
                        'processors': [
                            {'dissect': {'tokenizer': 'id=%{id} m=%{message}'}},
                        ],
                        'fields': {
                            'hello': 'world',
                        }
                    })

        uid = uuid.uuid4()
        # Insert some events into cloudwatch logs
        self.client.put_log_events(logGroupName=self.log_group_name,
                                   logStreamName=self.log_stream_name,
                                   logEvents=[
                                       {
                                           'timestamp': time_now_millis(),
                                           'message': 'id=%s m=hello world 1' % uid
                                       },
                                       {
                                           'timestamp': time_now_millis(),
                                           'message': 'id=%s m=hello world 2' % uid
                                       },
                                       {
                                           'timestamp': time_now_millis(),
                                           'message': 'id=%s m=hello world 3' % uid
                                       },
                                       {
                                           'timestamp': time_now_millis(),
                                           'message': 'id=%s m=hello world 4' % uid
                                       },
                                       {
                                           'timestamp': time_now_millis(),
                                           'message': 'id=%s m=hello world 5' % uid
                                       },
                                   ])

        self.wait_until(lambda: self.es.search(index="functionbeat*", body={"query": {"match_all": {}}})['hits']['total'] >= 5,
                        max_timeout=TIMEOUT)
        results = self.es.search(index="functionbeat*")
        messages = self.collect_messages(results)

        assert_equals(len(messages), 5)
        for i in range(0, 4):
            self.assertDictContainsSubset({
                'dissect': {
                    'id': str(uid),
                    'message': 'hello world %s' % (i+1),
                },
                'message': 'id=%s m=hello world %s' % (uid, (i+1)),
                'log_group': self.log_group_name,
                'fields': {
                    'hello': 'world',
                },
            }, messages[i])

        self.remove()

    @unittest.skipIf(not FUNCTIONBEAT_INTEGRATION_TESTS,
                     "functionbeat integration tests are disabled, run with FUNCTIONBEAT_INTEGRATION_TESTS=1 to enable them.")
    def test_deploy_cloudwatch_lambda_options(self):
        """
        Deploy a function and assert lambda options.
        """

        concurrency = 5
        timeout = 20
        description = 'The quick brown fox jumps over the lazy dog'

        self.deploy(deploy_bucket=self.bucket,
                    function={
                        'name': self.function_name,
                        'enabled': True,
                        'description': description,
                        'type': 'cloudwatch_logs',
                        'concurrency': concurrency,
                        'memory_size': '256MiB',
                        'timeout': timeout,
                        'triggers': [
                            {'log_group_name': self.log_group_name},
                        ],
                    },
                    cloud_id=self.cloud_id(),
                    cloud_auth=self.cloud_auth(),
                    )

        lbd = boto3.client('lambda')
        configs = lbd.get_function(FunctionName=self.function_name)

        # Assert runtime variables
        self.assertDictContainsSubset({
            'Variables': {
                'ENABLED_FUNCTIONS': self.function_name,
                'BEAT_STRICT_PERMS': 'false',
            },
        }, configs['Configuration']['Environment'])

        # Assert top level configuration limit.
        self.assertDictContainsSubset({
            'Timeout': timeout,
            'Description': description,
            'MemorySize': 256,
        }, configs['Configuration'])

        # Assert concurrency
        self.assertDictContainsSubset({
            'Concurrency': {
                'ReservedConcurrentExecutions': concurrency,
            },
        }, configs)

        self.remove()

    def deploy(self, **kw):
        # Setup the function
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*"
        )

        # Setup the function
        self.render_config_template(**kw)

        exit_code = self.run_beat(extra_args=["deploy", self.function_name])
        assert exit_code == 0

    def remove(self):
        exit_code = self.run_beat(extra_args=["remove", self.function_name])
        assert exit_code == 0


def time_now_millis():
    return long(time.time() * 1000)
