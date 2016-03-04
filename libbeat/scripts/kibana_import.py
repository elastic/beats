from elasticsearch import Elasticsearch
import argparse
import os

def store_object(es, type, name, doc):
    print es.index(index=".kibana", doc_type=type, id=name, body=doc)


def main():
    parser = argparse.ArgumentParser(
        description="Loads Kibana dashboards, vizualization and " +
                    "searches into Kibana")
    parser.add_argument("--url", help="Elasticsearch URL. E.g. " +
                                      "http://localhost:9200.", required=True)
    parser.add_argument("--dir", help="Input directory (kibana folder)", default="saved", required=True)

    args = parser.parse_args()

    es = Elasticsearch(args.url)

    base = args.dir
    folders = os.listdir(base)

    for folder in folders:

        base_dir = base + "/" + folder + "/"

        if os.path.isdir(base_dir):
            files = os.listdir(base_dir)

            for file in files:
                if os.path.isfile(base_dir + file) and os.path.splitext(file)[1] == '.json':
                    f = open(base_dir + file, 'r')
                    doc = f.read()

                    type = os.path.splitext(file)[0]

                    # Fixes windows problem with files with * inside
                    # Adds it to index pattern
                    if folder == "index-pattern":
                        type = type + "-*"
                    store_object(es, folder, type, doc)


if __name__ == "__main__":
    main()

