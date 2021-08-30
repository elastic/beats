import json
import sys
import glob
import argparse


def transform_data(data, method):
    if "attributes" not in data:
        return

    if "uiStateJSON" in data["attributes"]:
        data["attributes"]["uiStateJSON"] = method(data["attributes"]["uiStateJSON"])

    if "optionsJSON" in data["attributes"]:
        data["attributes"]["optionsJSON"] = method(data["attributes"]["optionsJSON"])

    if "panelsJSON" in data["attributes"]:
        data["attributes"]["panelsJSON"] = method(data["attributes"]["panelsJSON"])

    if "visState" in data["attributes"]:
        data["attributes"]["visState"] = method(data["attributes"]["visState"])

    if "kibanaSavedObjectMeta" in data["attributes"] and "searchSourceJSON" in data["attributes"]["kibanaSavedObjectMeta"]:
        data["attributes"]["kibanaSavedObjectMeta"]["searchSourceJSON"] = method(
            data["attributes"]["kibanaSavedObjectMeta"]["searchSourceJSON"])


def transform_file(path, method):
    with open(path, encoding='utf_8') as f:
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

        with open(path, 'w', encoding='utf_8') as f:
            f.write(new_data)
