import os
import json
import jinja2
import sys


def load_json(filepath):
    json_data=open(filepath).read()
    return json.loads(json_data)


def construct_dashboard(path, beatname):

    template = get_template("dashboard.json.j2")

    panelsJSON = load_escape_json_string("dashboard/" + path + "/panelsJSON.json")
    searchSourceJSON = load_escape_json_string("dashboard/" + path + "/searchSourceJSON.json")

    output_str = template.render(panelsJSON = panelsJSON, searchSourceJSON = searchSourceJSON, beatname=beatname)
    with open("./etc/kibana/dashboard/" + path + ".json", "wb") as f:
        f.write(output_str)

def construct_index_pattern(beatname):

    template = get_template("index-pattern.json.j2")

    fields = load_escape_json_string("index-pattern/" + beatname + "/fields.json")

    output_str = template.render(fields = fields, beatname=beatname)
    with open("./etc/kibana/index-pattern/" + beatname + ".json", "wb") as f:
        f.write(output_str)


def construct_search(path):
    template = get_template("search.json.j2")

    searchSourceJSON = load_escape_json_string("search/" + path + "/searchSourceJSON.json")

    output_str = template.render(searchSourceJSON = searchSourceJSON, title=path)
    with open("./etc/kibana/search/" + path + ".json", "wb") as f:
        f.write(output_str)

def construct_visualization():
    template = get_template("visualization.json.j2")

    base = './etc/kibana/visualization'

    files = os.listdir(base)
    for file in files :

        if os.path.isfile(base + "/" + file) and os.path.splitext(file)[1] == '.json':

            title = os.path.splitext(file)[0]

            searchSourceJSON = load_escape_json_string("visualization/" +  title + "/searchSourceJSON.json")
            visState = load_escape_json_string("visualization/" +  title + "/visState.json")

            output_str = template.render(searchSourceJSON = searchSourceJSON, visState=visState, title = title)
            with open("./etc/kibana/visualization/" + file, "wb") as f:
                f.write(output_str)


def get_template(name):
    # TODO: Fetch environment variables?
    template_path = '../libbeat/etc/kibana/'

    template_env = jinja2.Environment(
        loader=jinja2.FileSystemLoader(template_path)
    )

    template = template_env.get_template(name)

    return template

def load_escape_json_string(path):
    data = load_json("./etc/kibana/" + path)
    data = json.dumps(data, separators=(',',':'))
    data = data.replace('\\', '\\\\')
    data = data.replace('"', '\\"')
    return data

def construct():

    beatname = sys.argv[1]

    base = './etc/kibana'

    if os.path.exists(base + "/visualization"):
        visualization_dir = os.listdir(base + "/visualization")
        for file in visualization_dir :
            if os.path.isdir(base + "/visualization/" + file):
                construct_visualization()

    if os.path.exists(base + "/search"):
        search_dir = os.listdir(base + "/search")
        for file in search_dir :
            if os.path.isdir(base + "/search/" + file):
                construct_search(file)


    if os.path.exists(base + "/dashboard"):
        dashboard_dir = os.listdir(base + "/dashboard")
        for file in dashboard_dir :
            if os.path.isdir(base + "/dashboard/" + file):
                # TODO: Get from environment
                construct_dashboard(file, "Topbeat")

    if os.path.exists(base + "/index-pattern"):
        index_pattern_dir = os.listdir(base + "/index-pattern")
        for file in index_pattern_dir :
            if os.path.isdir(base + "/index-pattern/" + file):
                construct_index_pattern(beatname)


construct()
