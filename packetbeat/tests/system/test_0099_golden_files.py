import json
import os
import yaml
from packetbeat import BaseTest
from parameterized import parameterized


current_dir = os.path.dirname(os.path.abspath(__file__))
golden_dir = os.path.join(current_dir, 'golden')
pcaps_dir = os.path.join(current_dir, 'pcaps')
config_file = os.path.join(current_dir, "config/golden-tests.yml")
golden_suffix = '-expected.json'


def load_golden_test_cases():
    """
    Loads the test cases from config/golden-tests.yml
    """
    repl_chars = ' -.'
    cases = []
    with open(config_file, 'r') as stream:
        yml = yaml.safe_load(stream)
        for case in yml['test_cases']:
            name = ''.join(['_' if c in repl_chars else c for c in case['name']]).lower()
            cases.append((name,
                          os.path.join(current_dir, case['pcap']),
                          case['config']))
    return cases


class Test(BaseTest):

    @parameterized.expand(load_golden_test_cases)
    def test_golden_files(self, name, pcap, config):
        golden = os.path.join(golden_dir, name + golden_suffix)
        self.render_config_template(**config)
        self.run_packetbeat(pcap=pcap)
        objs = [self.flatten_object(clean_keys(o), {}, "") for o in self.read_output()]
        assert len(objs) > 0, "No output generated"

        if os.getenv("GENERATE"):
            with open(os.path.join(golden_dir, golden), 'w') as f:
                json.dump(objs, f, indent=4, separators=(',', ': '), sort_keys=True)

        with open(os.path.join(golden_dir, golden), 'r') as f:
            expected = json.load(f)

        assert len(expected) == len(objs), "expected {} events to compare but got {}".format(
            len(expected), len(objs))

        for ev in expected:
            clean_keys(ev)
            found = False
            for obj in objs:
                if ev == obj:
                    found = True
                    break
            assert found, "The following expected object was not found:\n {}\nSearched in: \n{}".format(
                pretty_json(ev), pretty_json(objs))


def clean_keys(obj):
    # These keys are host dependent
    keys = [
        "agent.ephemeral_id",
        "agent.hostname",
        "agent.id",
        "agent.type",
        "agent.version",
        "ecs.version",
        "event.end",
        "event.start",
        "host.name",

        # Remove when Packetbeat can use the timestamp from pcap files.
        "@timestamp",

        # Network direction is populated based on local-IPs which is misleading
        # when reading from a pcap and leads to inconsistent results.
        "network.direction",
    ]
    for key in keys:
        delete_key(obj, key)
    return obj


def delete_key(obj, key):
    if key in obj:
        del obj[key]


def pretty_json(obj):
    return json.dumps(obj, indent=2, separators=(',', ': '))
