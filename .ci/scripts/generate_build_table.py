#!/usr/bin/env python3

import os
import yaml

if __name__ == "__main__":

    print("| Beat  | Stage  | Command  | MODULE  | Platforms  | When |")
    print("|-------|--------|----------|---------|------------|------|")
    for root, dirs, files in os.walk("."):
        dirs.sort()
        for file in files:
            if file.endswith("Jenkinsfile.yml") and root != ".":
                with open(os.path.join(root, file), 'r') as f:
                    doc = yaml.load(f, Loader=yaml.FullLoader)
                module = root.replace(".{}".format(os.sep), '')
                platforms = [doc["platform"]]
                when = ""
                if "branches" in doc["when"]:
                    when = f"{when}/:palm_tree:"
                if "changeset" in doc["when"]:
                    when = f"{when}/:file_folder:"
                if "comments" in doc["when"]:
                    when = f"{when}/:speech_balloon:"
                if "labels" in doc["when"]:
                    when = f"{when}/:label:"
                if "parameters" in doc["when"]:
                    when = f"{when}/:smiley:"
                if "tags" in doc["when"]:
                    when = f"{when}/:taco:"
                for stage in doc["stages"]:
                    withModule = False
                    if "make" in doc["stages"][stage]:
                        command = doc["stages"][stage]["make"]
                    if "mage" in doc["stages"][stage]:
                        command = doc["stages"][stage]["mage"]
                    if "platforms" in doc["stages"][stage]:
                        platforms = doc["stages"][stage]["platforms"]
                    if "withModule" in doc["stages"][stage]:
                        withModule = doc["stages"][stage]["withModule"]
                    if "when" in doc["stages"][stage]:
                        when = f"{when}/:star:"
                    print("| {} | {} | `{}` | {} | `{}` | {} |".format(
                        module, stage, command, withModule, platforms, when))

print("> :palm_tree: -> Git Branch based")
print("> :label: -> GitHub Pull Request Label based")
print("> :file_folder: -> Changeset based")
print("> :speech_balloon: -> GitHub Pull Request comment based")
print("> :taco: -> Git tag based")
print("> :smiley: -> Manual UI interaction based")
print("> :star: -> More specific cases based")
