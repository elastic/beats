import os, re


modules = os.listdir()
doc = os.path.realpath(os.path.join(os.getcwd(), "..", "..", "..", "docs", "reference", "metricbeat"))

for module in modules:
    if not os.path.isdir(module):
        continue
    metricsets = os.listdir(module)
    for metricset in metricsets:
        if (not os.path.isdir(os.path.join(module, metricset))) or metricset == "_meta":
            continue
        name = "metricbeat-metricset-{}-{}".format(module, metricset)
        if os.path.exists(os.path.join(doc, name+".md")):
            with open(os.path.join(doc,  name+".md"), "r") as f:
                data = f.read()
                regex = r"\[{}\]\n+((?:.+\n+)+)\n+(?=## Fields)".format(name)
                found = re.search(regex, data)
                if found:
                    with open(os.path.join(module, metricset, "_meta", "docs.md"), "w") as f2:
                        f2.write(found.group(1))
                    print("Found and wrote: {}".format(os.path.join(module, metricset, "_meta", "docs.asciidoc")))
                    try:
                        os.remove(os.path.join(module, metricset, "_meta", "docs.asciidoc"))
                    except:
                        pass
                else:
                    print("Not found: {}".format(name))
                        