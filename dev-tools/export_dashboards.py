from elasticsearch import Elasticsearch
import argparse
import os
import json
import re


def ExportDashboards(es, beat, kibana_index, output_directory):
    res = es.search(
        index=kibana_index,
        doc_type="dashboard",
        size=1000)

    reg_exp = re.compile(beat + '*', re.IGNORECASE)

    for doc in res['hits']['hits']:

        if not reg_exp.match(doc["_source"]["title"]):
            print("Ignore dashboard", doc["_source"]["title"])
            continue

        # save dashboard
        SaveJson("dashboard", doc, output_directory)

        # save dependencies
        panels = json.loads(doc['_source']['panelsJSON'])
        for panel in panels:
            if panel["type"] == "visualization":
                ExportVisualization(
                    es,
                    panel["id"],
                    kibana_index,
                    output_directory)
            elif panel["type"] == "search":
                ExportSearch(
                    es,
                    panel["id"],
                    kibana_index,
                    output_directory)
            else:
                print("Unknown type {} in dashboard".format(panel["type"]))


def ExportVisualization(es, visualization, kibana_index, output_directory):
    doc = es.get(
        index=kibana_index,
        doc_type="visualization",
        id=visualization)

    # save visualization
    SaveJson("visualization", doc, output_directory)

    # save dependencies
    if "savedSearchId" in doc["_source"]:
        search = doc["_source"]['savedSearchId']
        ExportSearch(
            es,
            search,
            kibana_index,
            output_directory)


def ExportSearch(es, search, kibana_index, output_directory):
    doc = es.get(
        index=kibana_index,
        doc_type="search",
        id=search)

    # save search
    SaveJson("search", doc, output_directory)


def ExportIndex(es, index, kibana_index, output_directory):
    doc = es.get(
        index=kibana_index,
        doc_type="index-pattern",
        id=index)

    # Fixes windows problem with files with * inside
    # Removes it from index pattern
    doc['_id'] = doc['_id'][:-2]

    # save index-pattern
    SaveJson("index-pattern", doc, output_directory)


def SaveJson(doc_type, doc, output_directory):

    dir = os.path.join(output_directory, doc_type)
    if not os.path.exists(dir):
        os.makedirs(dir)

    filepath = os.path.join(dir, doc['_id'] + '.json')
    with open(filepath, 'w') as f:
        json.dump(doc['_source'], f, indent=2)
        print("Written {}".format(filepath))


def main():
    parser = argparse.ArgumentParser(
        description="Export the Kibana dashboards together with"
                    " all used visualizations, searches and index pattern")
    parser.add_argument("--url",
                        help="Elasticsearch URL. E.g. http://localhost:9200",
                        default="http://localhost:9200")
    parser.add_argument("--beat",
                        help="Beat name e.g. topbeat",
                        required=True)
    parser.add_argument("--index",
                        help="Elasticsearch index for the Beat data. "
                        "E.g. topbeat-*")
    parser.add_argument("--kibana",
                        help="Elasticsearch index for the Kibana dashboards. "
                        "E.g. .kibana",
                        default=".kibana")
    parser.add_argument("--dir", help="Output directory. E.g. output",
                        default="output")

    args = parser.parse_args()

    if args.index is None:
        args.index = args.beat.lower() + "-*"

    print("Export {} dashboards to {} directory".format(args.beat, args.dir))
    print("Elasticsearch URL: {}".format(args.url))
    print("Elasticsearch index to store Beat's data: {}".format(args.index))
    print("Elasticsearch index to store Kibana's"
          " dashboards: {}".format(args.kibana))

    es = Elasticsearch(args.url)
    ExportIndex(es, args.index, args.kibana, args.dir)
    ExportDashboards(es, args.beat, args.kibana, args.dir)

if __name__ == "__main__":
    main()
