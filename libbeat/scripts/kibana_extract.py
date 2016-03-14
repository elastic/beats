
import os
import json
import jinja2


def unescape_json(escaped):
    obj = json.loads(escaped)
    return json.dumps(obj, sort_keys=True, indent=4, separators=(',', ': '))

def load_json(filepath):
    json_data=open(filepath).read()
    return json.loads(json_data)

def load_search_source(file):
    data = load_json(file)

    field = "searchSourceJSON"

    if data.has_key("kibanaSavedObjectMeta") and data["kibanaSavedObjectMeta"].has_key(field):
        jsonData = unescape_json(data["kibanaSavedObjectMeta"][field])
    else:
        return ""

    write_output(file, field, jsonData)


def load_field(file, field):
    data = load_json(file)

    if  data.has_key(field):
        jsonData = unescape_json(data[field])
    else:
        return ""

    write_output(file, field, jsonData)


def write_output(file, field, jsonData):

    dir_name = os.path.splitext(file)[0]

    if not os.path.exists(dir_name):
        os.mkdir(dir_name)

    with open(dir_name + "/" + field + ".json", 'w') as outfile:
        outfile.write(jsonData)


def extract():
    base = './etc/kibana'

    folders = os.listdir(base)

    for folder in folders:
        base_dir = base + "/" + folder + "/"
        if os.path.isdir(base_dir):

            files = os.listdir(base_dir)
            for file in files :
                # Only json files
                if os.path.isfile(base_dir + file) and os.path.splitext(file)[1] == '.json':

                    load_search_source(base_dir + file)
                    load_field(base_dir + file, "visState")
                    load_field(base_dir + file, "fields")
                    load_field(base_dir + file, "fieldFormatMap")
                    load_field(base_dir + file, "panelsJSON")


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

extract()

