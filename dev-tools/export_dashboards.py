from elasticsearch import Elasticsearch
import argparse
import os
import json
import re


def ExportDashboards(es, regex, kibana_index, output_directory):
    res = es.search(
        index=kibana_index,
        doc_type="dashboard",
        size=1000)

    try:
        reg_exp = re.compile(regex, re.IGNORECASE)
    except:
        print("Wrong regex {}".format(regex))
        return

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


def SaveJson(doc_type, doc, output_directory):

    dir = os.path.join(output_directory, doc_type)
    if not os.path.exists(dir):
        os.makedirs(dir)
    # replace unsupported characters
    filepath = os.path.join(dir, re.sub(r'[\>\<:"/\\\|\?\*]', '', doc['_id']) + '.json')
    with open(filepath, 'w') as f:
        json.dump(doc['_source'], f, indent=2)
        print("Written {}".format(filepath))


def main():
    parser = argparse.ArgumentParser(
        description="Export the Kibana dashboards together with"
                    " all used visualizations, searches and index pattern")
    parser.add_argument("--url",
                        help="Elasticsearch URL. By default: http://localhost:9200",
                        default="http://localhost:9200")
    parser.add_argument("--regex",
                        help="Regular expression to match all the dashboards to be exported. For example: metricbeat*",
                        required=True)
    parser.add_argument("--kibana",
                        help="Elasticsearch index where to store the Kibana settings. By default: .kibana ",
                        default=".kibana")
    parser.add_argument("--dir", help="Output directory. By default: output",
                        default="output")

    args = parser.parse_args()

    print("Export {} dashboards to {} directory".format(args.regex, args.dir))
    print("Elasticsearch URL: {}".format(args.url))
    print("Elasticsearch index to store Kibana's"
          " dashboards: {}".format(args.kibana))

    es = Elasticsearch(args.url)
    ExportDashboards(es, args.regex, args.kibana, args.dir)

if __name__ == "__main__":
    main()
