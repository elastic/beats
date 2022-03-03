#!/usr/bin/env python3

import os
import yaml

if __name__ == "__main__":

    print("| Beat  | Stage  | Category | Command  | MODULE  | Platforms  | When |")
    print("|-------|--------|----------|----------|---------|------------|------|")
    for root, dirs, files in os.walk("."):
        dirs.sort()
        for file in files:
            if file.endswith("Jenkinsfile.yml") and root != ".":
                with open(os.path.join(root, file), 'r') as f:
                    doc = yaml.load(f, Loader=yaml.FullLoader)
                module = root.replace(".{}".format(os.sep), '')
                for stage in doc["stages"]:
                    withModule = False
                    platforms = [doc["platform"]]
                    when = "mandatory"
                    category = 'default'
                    if "stage" in doc["stages"][stage]:
                        category = doc["stages"][stage]["stage"]
                    if "make" in doc["stages"][stage]:
                        command = doc["stages"][stage]["make"].replace("\n", " ")
                    if "mage" in doc["stages"][stage]:
                        command = doc["stages"][stage]["mage"].replace("\n", " ")
                    if "k8sTest" in doc["stages"][stage]:
                        command = doc["stages"][stage]["k8sTest"]
                    if "cloud" in doc["stages"][stage]:
                        command = doc["stages"][stage]["cloud"]
                    if "platforms" in doc["stages"][stage]:
                        platforms = doc["stages"][stage]["platforms"]
                    if "withModule" in doc["stages"][stage]:
                        withModule = doc["stages"][stage]["withModule"]
                    if "when" in doc["stages"][stage]:
                        if "not_changeset_full_match" not in doc["stages"][stage]["when"]:
                            when = "optional"
                    print("| {} | {} | `{}` | `{}` | {} | `{}` | {} |".format(
                        module, stage, category, command, withModule, platforms, when))
