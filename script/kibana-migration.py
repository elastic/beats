import yaml
import glob


def migration():
    print "Start Kibana files migration"

    print "Migrate all fields to the ECS fields"
    migration_fields = read_migration_fields()
    rename_entries(migration_fields)

    print "Postfix all ids with -ecs"
    ids = get_replaceable_ids()
    rename_entries(ids)

    print "Postfix all titles with ` ECS`"
    titles = get_replacable_titles()
    rename_entries(titles)


def get_replaceable_ids():
    files = get_files()

    ids = {}
    for file in files:
        with open(file, 'r') as f:
            objects = yaml.safe_load(f)

            for v in objects["objects"]:
                # Checks if an id was already migrated, if not adds it to the list
                if "-ecs" not in v["id"]:
                    # Add "{}" around fields to make them more unique and not have false positives
                    ids['"' + v["id"] + '"'] = '"' + v["id"] + "-ecs" + '"'
                    # Prefix with / to also modify links
                    ids['/' + v["id"]] = '/' + v["id"] + "-ecs"

    return ids


def read_migration_fields():
    migration_fields = {}
    migration_yml = "../dev-tools/ecs-migration.yml"
    with open(migration_yml, 'r') as f:
        migration = yaml.safe_load(f)
        for k in migration:
            if "to" in k and "from" in k:
                if "rename" in k and k["rename"] is False:
                    continue
                if k["alias"] == False:
                    continue
                if not isinstance(k["to"], basestring):
                    continue

                # Add "{}" around fields to make them more unique and not have false positives
                migration_fields['"' + k["from"] + '"'] = '"' + k["to"] + '"'
                # Some fields exist inside a query / filter where they are followed by :
                migration_fields[k["from"] + ':'] = k["to"] + ':'

    return migration_fields


def get_replacable_titles():
    files = get_files()

    titles = {}
    for file in files:
        with open(file, 'r') as f:
            objects = yaml.safe_load(f)

            for v in objects["objects"]:

                # Add "{}" around titles to make them more unique and not have false positives
                if "title" in v["attributes"]:
                    if "ECS" not in v["attributes"]["title"]:
                        titles['"' + v["attributes"]["title"] + '"'] = '"' + v["attributes"]["title"] + " ECS" + '"'

                if "visState" in v["attributes"] and "title" in v["attributes"]["visState"]:
                    if "ECS" not in v["attributes"]["visState"]["title"]:
                        titles['"' + v["attributes"]["visState"]["title"] + '"'] = '"' + \
                            v["attributes"]["visState"]["title"] + " ECS" + '"'

    return titles


def rename_entries(renames):
    files = get_files()

    for file in files:
        print file
        s = open(file).read()
        for k in renames:
            s = s.replace(k, renames[k])
        f = open(file, 'w')
        f.write(s)
        f.close()


def get_files():
    all_beats = '../*/_meta/kibana/7/dashboard/*.json'
    module_beats = '../*/module/*/_meta/kibana/7/dashboard/*.json'
    heartbeat = '../heartbeat/monitors/active/*/_meta/kibana/7/dashboard/*.json'
    xpack_module_beats = '../x-pack/*/module/*/_meta/kibana/7/dashboard/*.json'

    return glob.glob(all_beats) + glob.glob(module_beats) + glob.glob(heartbeat) + glob.glob(xpack_module_beats)


if __name__ == "__main__":
    migration()


# There are more id's, do they matter?
# Example:
#
# "series": [
#    {
#        "axis_position": "right",
#        "chart_type": "line",
#        "color": "#68BC00",
#        "fill": 0.5,
#        "formatter": "number",
#        "id": "6984af11-4d5d-11e7-aa29-87a97a796de6",
#        "label": "In Packetloss",
#        "line_width": 1,
#        "metrics": [
#            {
#                "field": "system.network.in.dropped",
#                "id": "6984af12-4d5d-11e7-aa29-87a97a796de6",
#                "type": "max"
#            }
#        ],
