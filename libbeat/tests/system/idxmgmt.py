from elasticsearch import NotFoundError
from nose.tools import raises
import datetime


class IdxMgmt(object):

    def __init__(self, client, index):
        self._client = client
        self._index = index if index != '' and index != '*' else 'mockbeat'

    def needs_init(self, s):
        return s == '' or s == '*'

    def delete(self, indices=[]):
        indices = list(filter(lambda x: x != '', indices))
        if not indices:
            indices == [self._index]
        for i in indices:
            self.delete_index_and_alias(i)
            self.delete_template(template=i)
        for i in indices:
            self.delete_policy(policy=i)

    def delete_index_and_alias(self, index=""):
        if self.needs_init(index):
            index = self._index

        try:
            self._client.transport.perform_request('DELETE', "/" + index + "*")
        except NotFoundError:
            pass

    def delete_template(self, template=""):
        if self.needs_init(template):
            template = self._index

        try:
            self._client.transport.perform_request('DELETE', "/_template/" + template + "*")
        except NotFoundError:
            pass

    def delete_policy(self, policy=""):
        if self.needs_init(policy):
            policy = self._index

        # Delete any existing policy starting with given policy
        policies = self._client.transport.perform_request('GET', "/_ilm/policy")
        for p, _ in policies.items():
            if not p.startswith(policy):
                continue
            try:
                self._client.transport.perform_request('DELETE', "/_ilm/policy/" + p)
            except NotFoundError:
                pass

    @raises(NotFoundError)
    def assert_index_template_not_loaded(self, template):
        self._client.transport.perform_request('GET', '/_template/' + template)

    def assert_index_template_loaded(self, template):
        resp = self._client.transport.perform_request('GET', '/_template/' + template)
        assert template in resp
        assert "lifecycle" not in resp[template]["settings"]["index"]

    def assert_ilm_template_loaded(self, template, policy, alias):
        resp = self._client.transport.perform_request('GET', '/_template/' + template)
        assert resp[template]["settings"]["index"]["lifecycle"]["name"] == policy
        assert resp[template]["settings"]["index"]["lifecycle"]["rollover_alias"] == alias

    def assert_index_template_index_pattern(self, template, index_pattern):
        resp = self._client.transport.perform_request('GET', '/_template/' + template)
        assert template in resp
        assert resp[template]["index_patterns"] == index_pattern

    def assert_alias_not_created(self, alias):
        resp = self._client.transport.perform_request('GET', '/_alias')
        for name, entry in resp.items():
            if alias not in name:
                continue
            assert entry["aliases"] == {}, entry["aliases"]

    def assert_alias_created(self, alias, pattern=None):
        if pattern is None:
            pattern = self.default_pattern()
        name = alias + "-" + pattern
        resp = self._client.transport.perform_request('GET', '/_alias/' + alias)
        assert name in resp
        assert resp[name]["aliases"][alias]["is_write_index"] == True

    @raises(NotFoundError)
    def assert_policy_not_created(self, policy):
        self._client.transport.perform_request('GET', '/_ilm/policy/' + policy)

    def assert_policy_created(self, policy):
        resp = self._client.transport.perform_request('GET', '/_ilm/policy/' + policy)
        assert policy in resp
        assert resp[policy]["policy"]["phases"]["hot"]["actions"]["rollover"]["max_size"] == "50gb"
        assert resp[policy]["policy"]["phases"]["hot"]["actions"]["rollover"]["max_age"] == "30d"

    def assert_docs_written_to_alias(self, alias, pattern=None):
        if pattern is None:
            pattern = self.default_pattern()
        name = alias + "-" + pattern
        data = self._client.transport.perform_request('GET', '/' + name + '/_search')
        assert data["hits"]["total"] > 0

    def default_pattern(self):
        d = datetime.datetime.now().strftime("%Y.%m.%d")
        return d + "-000001"

    def index_for(self, alias, pattern=None):
        if pattern is None:
            pattern = self.default_pattern()
        return "{}-{}".format(alias, pattern)
