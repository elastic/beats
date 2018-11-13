import yaml


def to_yaml(input, indent=4, **kw):
    return yaml.dump(input, indent=indent, default_flow_style=False, **kw)
