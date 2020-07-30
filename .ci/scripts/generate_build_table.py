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
                platforms = [ doc["platform"] ]
                for stage in doc["stages"]:
                    withModule = False
                    when = "Always"
                    if "make" in doc["stages"][stage]:
                        command = doc["stages"][stage]["make"]
                    if "mage" in doc["stages"][stage]:
                        command = doc["stages"][stage]["mage"]
                    if "platforms" in doc["stages"][stage]:
                        platforms = doc["stages"][stage]["platforms"]
                    if "withModule" in doc["stages"][stage]:
                        withModule = doc["stages"][stage]["withModule"]
                    if "when" in doc["stages"][stage]:
                        when = "some cases"
                    print("| {} | {} | `{}` | {} | `{}` | `{}` |".format(module, stage, command, withModule, platforms, when))
