import json
import sys
import glob
import argparse


def transform_data(data, method):
    for obj in data["objects"]:
        if "uiStateJSON" in obj["attributes"]:
            obj["attributes"]["uiStateJSON"] = method(obj["attributes"]["uiStateJSON"])

        if "optionsJSON" in obj["attributes"]:
            obj["attributes"]["optionsJSON"] = method(obj["attributes"]["optionsJSON"])

        if "panelsJSON" in obj["attributes"]:
            obj["attributes"]["panelsJSON"] = method(obj["attributes"]["panelsJSON"])

        if "visState" in obj["attributes"]:
            obj["attributes"]["visState"] = method(obj["attributes"]["visState"])

        if "kibanaSavedObjectMeta" in obj["attributes"] and "searchSourceJSON" in obj["attributes"]["kibanaSavedObjectMeta"]:
            obj["attributes"]["kibanaSavedObjectMeta"]["searchSourceJSON"] = method(
                obj["attributes"]["kibanaSavedObjectMeta"]["searchSourceJSON"])


def transform_file(path, method):
    with open(path) as f:
        data = json.load(f)

    transform_data(data, method)
    return data


if __name__ == "__main__":

    parser = argparse.ArgumentParser(description="Convert dashboards")
    parser.add_argument("--transform", help="Decode or encode", default="encode")
    parser.add_argument("--glob", help="Glob pattern")

    args = parser.parse_args()

    paths = glob.glob(args.glob)

    method = json.dumps
    if args.transform == "decode":
        method = json.loads

    for path in paths:
        data = transform_file(path, method)
        new_data = json.dumps(data, sort_keys=True, indent=4)

        with open(path, 'w') as f:
            f.write(new_data)
