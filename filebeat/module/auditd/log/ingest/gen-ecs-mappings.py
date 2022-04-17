#!/usr/bin/env python3

# This script generates auditd to ECS mappings from github.com/menderesk/go-libaudit
#
# Usage: ./gen-ecs-mappings.py ~/go/src/github.com/menderesk/go-libaudit
#
# It will output to stdout the `params` section for the script processor in the ingest pipeline.
import copy
import os
import sys
import yaml
from collections import defaultdict
from shlex import quote
from subprocess import check_call, call, check_output


def extract_object(name: str, source: dict) -> dict:
    r = {}
    for k, v in source.items():
        if k == 'primary' or k == 'secondary':
            r[name + '.' + k] = v
        elif k == 'what' or k == 'path_index' or k == 'how':
            pass
        else:
            raise Exception('Unexpected object key: ' + k)
    return r


def map_object(instance: dict, context: str, mappings: dict):
    for k, v in instance.items():
        if k not in mappings:
            raise Exception('Unexpected key "{}" while parsing {}'.format(k, context))
        mappings[k](k, v)


def convert_mappings(m: dict) -> dict:
    event = {}
    objects = {
        # Default values for subject (actor), may be overridden.
        'subject.primary': ['auid'],
        'subject.secondary': ['uid'],
    }
    extra = {}  # TODO: Unused (sets client.ip)
    mappings = []
    has_fields = []

    def store_condition(k: str, v: list):
        nonlocal has_fields
        has_fields = v

    def store_event(k: str, v: list):
        if not isinstance(v, list):
            v = [v]
        event[k] = v

    def ignore(k, v):
        pass

    def make_store_field(name: str):
        def store(k: str, v: any):
            extra[name] = v
        return store

    def store_ecs(k: str, v: dict):
        def store_mappings(k: str, v: list):
            if not isinstance(v, list):
                raise Exception('ecs.mappings must be a list, not ' + repr(v))
            nonlocal mappings
            mappings = v

        map_object(v, 'ecs', {
            'type': store_event,
            'category': store_event,
            'mappings': store_mappings,
        })

    def store_entity(basek: str, basev: dict):
        def save(k: str, v: any):
            if not isinstance(v, list):
                v = [v]
            objects[basek + '.' + k] = v

        map_object(basev, basek, {
            **dict.fromkeys(['primary', 'secondary'], save),
            **dict.fromkeys(['what', 'path_index'], ignore)
        })

    map_object(m, 'mapping', {
        'action': store_event,
        'ecs': store_ecs,
        'source_ip': make_store_field('source.ip'),
        'has_fields': store_condition,
        **dict.fromkeys(['object', 'subject'], store_entity),
        **dict.fromkeys(['syscalls', 'record_types', 'how', 'description'], ignore),
    })
    d = {
        'event': event,
    }

    if len(mappings) > 0:
        d['copy'] = []
        for mp in mappings:
            ref = mp['from']
            if ref in objects:
                source = objects[ref]
            else:
                parts = ref.split('.')
                if len(parts) != 2:
                    raise Exception("Don't know how to apply ecs mapping for {}".format(ref))
                if parts[0] == 'uid' or parts[0] == 'data':
                    source = [parts[1]]
                else:
                    raise Exception("Don't know how to apply ecs mapping for {}".format(ref))
            d['copy'].append({
                'from': source,
                'to': mp['to']
            })

    if len(has_fields) > 0:
        d['has_fields'] = has_fields
    return d


class DefaultDict(defaultdict):
    def __init__(self, factory):
        super(DefaultDict, self).__init__(factory)

    def append(self, keys, obj):
        if isinstance(keys, str):
            keys = [keys]
        for key in keys:
            self[key].append(copy.deepcopy(obj))


if __name__ == '__main__':
    if len(sys.argv) != 2:
        print('Usage: {} <path/to/go-libaudit-repo>'.format(sys.argv[0]))
        sys.exit(1)
    repo_path = sys.argv[1]
    if not os.path.isdir(repo_path):
        raise Exception('Path to go-libaudit is not a directory: ' + repo_path)
    git_path = repo_path + "/.git"
    if not os.path.isdir(git_path):
        raise Exception('go-libaudit directory doesn\'t contain a git repository: ' + git_path)
    norms_path = repo_path + "/aucoalesce/normalizations.yaml"
    if not os.path.isfile(norms_path):
        raise Exception('go-libaudit repository doesn\'t contain the normalizations file: ' + norms_path)
    revision = check_output('git --work-tree={} --git-dir={} describe --tags'.format(quote(repo_path),
                                                                                     quote(git_path)), shell=True).decode('utf8').strip()
    with open(norms_path, 'r') as f:
        norms = yaml.full_load(f)
        types = DefaultDict(list)
        syscalls = DefaultDict(list)
        for entry in norms['normalizations']:
            proto = convert_mappings(entry)
            # TODO: Correctly check for emptyness (condition field?)
            if len(proto) == 0:
                continue
            if 'syscalls' in entry:
                syscalls.append(entry['syscalls'], proto)

            if 'record_types' in entry:
                types.append(entry['record_types'], proto)

if 'SYSCALL' in types:
    raise Exception('SYSCALL cannot be specified in record_types')

print('# Auditd record type to ECS mappings')
print('# AUTOGENERATED FROM go-libaudit {}, DO NOT EDIT'.format(revision))
yaml.safe_dump({
    'params': {
        'types': dict(types),
        'syscalls': dict(syscalls),
    }
}, sys.stdout)
print('# END OF AUTOGENERATED')
