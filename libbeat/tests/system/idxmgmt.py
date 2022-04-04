import datetime
import unittest
import pytest
from elasticsearch import NotFoundError


class IdxMgmt(unittest.TestCase):

    def __init__(self, client, index):
        self._client = client
        self._index = index if index != '' and index != '*' else 'mockbeat'
        self.patterns = [self.default_pattern(), "1", datetime.datetime.now().strftime("%Y.%m.%d")]

    def needs_init(self, s):
        return s == '' or s == '*'

    def delete(self, indices=[], policies=[], data_streams=[]):
        for ds in data_streams:
            self.delete_data_stream(ds)
            self.delete_template(template=ds)
        for i in indices:
            self.delete_index_and_alias(i)
            self.delete_template(template=i)
        for i in [x for x in policies if x != '']:
            self.delete_policy(i)

    def delete_data_stream(self, data_stream):
        try:
            resp = self._client.transport.perform_request('DELETE', '/_data_stream/' + data_stream)
        except NotFoundError:
            pass

    def delete_index_and_alias(self, index=""):
        if self.needs_init(index):
            index = self._index

        for pattern in self.patterns:
            index_with_pattern = index+"-"+pattern
            try:
                self._client.indices.delete(index_with_pattern)
                self._client.indices.delete_alias(index, index_with_pattern)
            except NotFoundError:
                continue

    def delete_template(self, template=""):
        if self.needs_init(template):
            template = self._index

        try:
            self._client.transport.perform_request('DELETE', "/_index_template/" + template)
        except NotFoundError:
            pass

    def delete_policy(self, policy):
        # Delete any existing policy starting with given policy
        policies = self._client.transport.perform_request('GET', "/_ilm/policy")
        for p, _ in policies.items():
            if not p.startswith(policy):
                continue
            try:
                self._client.transport.perform_request('DELETE', "/_ilm/policy/" + p)
            except NotFoundError:
                pass

    def assert_index_template_not_loaded(self, template):
        with pytest.raises(NotFoundError):
            self._client.transport.perform_request('GET', '/_index_template/' + template)

    def assert_index_template_loaded(self, template):
        resp = self._client.transport.perform_request('GET', '/_index_template/' + template)
        found = False
        for index_template in resp['index_templates']:
            if index_template['name'] == template:
                found = True
        assert found

    def assert_data_stream_created(self, data_stream):
        try:
            resp = self._client.transport.perform_request('GET', '/_data_stream/' + data_stream)
        except NotFoundError:
            assert False

    def assert_index_template_index_pattern(self, template, index_pattern):
        resp = self._client.transport.perform_request('GET', '/_index_template/' + template)
        for index_template in resp['index_templates']:
            if index_template['name'] == template:
                assert index_pattern == index_template['index_template']['index_patterns']
                found = True
        assert found

    def assert_policy_not_created(self, policy):
        with pytest.raises(NotFoundError):
            self._client.transport.perform_request('GET', '/_ilm/policy/' + policy)

    def assert_policy_created(self, policy):
        resp = self._client.transport.perform_request('GET', '/_ilm/policy/' + policy)
        assert policy in resp
        assert resp[policy]["policy"]["phases"]["hot"]["actions"]["rollover"]["max_size"] == "50gb"
        assert resp[policy]["policy"]["phases"]["hot"]["actions"]["rollover"]["max_age"] == "30d"

    def assert_docs_written_to_data_stream(self, data_stream):
        # Refresh the indices to guarantee all documents are available
        # through the _search API.
        self._client.transport.perform_request('POST', '/_refresh')

        data = self._client.transport.perform_request('GET', '/' + data_stream + '/_search')
        self.assertGreater(data["hits"]["total"]["value"], 0)

    def default_pattern(self):
        d = datetime.datetime.now().strftime("%Y.%m.%d")
        return d + "-000001"
