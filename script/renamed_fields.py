import yaml


def migration():

    beats = ["Auditbeat", "Filebeat", "Heartbeat", "Journalbeat", "Metricbeat", "Packetbeat", "Winlogbeat"]

    for beat in beats:
        print ".{} renamed fields in 7.0".format(beat)
        migration_fields = read_migration_fields(beat.lower())
        print get_table(migration_fields)


def get_table(migration_fields):
    out = """[frame="topbot",options="header"]
|======================
|Old Field|New Field
"""

    for k in migration_fields:
        out += '|`{}`            |`{}`\n'.format(k[0], k[1])

    out += "|======================\n"

    return out


def read_migration_fields(beat):
    migration_fields = {}
    migration_yml = "../dev-tools/ecs-migration.yml"
    with open(migration_yml, 'r') as f:
        migration = yaml.safe_load(f)
        for k in migration:
            if "beat" not in k or k["beat"] == beat:
                if "to" in k and "from" in k:
                    if not isinstance(k["to"], basestring):
                        continue
                    migration_fields[k["from"]] = k["to"]

    return sorted(migration_fields.items(), key=lambda x: x[0])


if __name__ == "__main__":
    migration()
