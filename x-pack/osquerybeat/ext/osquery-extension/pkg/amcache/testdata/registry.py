import argparse
import sqlite3
from tkinter.font import names

from Registry import Registry

def parse_arguments():
    parser = argparse.ArgumentParser(description="Process some registry data.")
    parser.add_argument("input", help="Input registry file")
    return parser.parse_args()


def create_database(db_path):
    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()
    cursor.execute('''
        CREATE TABLE IF NOT EXISTS InventoryApplicationFile (
            id INTEGER PRIMARY KEY,
            ProgramId TEXT,
            FileId TEXT,
            LowerCaseLongPath TEXT,
            Name TEXT,
            OriginalFileName TEXT,
            Publisher TEXT,
            Version TEXT,
            BinFileVersion TEXT,
            BinaryType TEXT,
            ProductName TEXT,
            ProductVersion TEXT,
            LinkDate TEXT,
            BinProductVersion TEXT,
        )
    ''')
    conn.commit()
    return conn

def create_table_statement(table_name, columns):
    column_definitions = ",\n".join([f"    {col.name} {col.data_type}" for col in columns])
    return f'''
    CREATE TABLE IF NOT EXISTS {table_name} (
{column_definitions}
    )
    '''


def main():
    args = parse_arguments()
    reg = Registry.Registry(args.input)

    keys = [
        "InventoryApplicationFile",
        "InventoryDriverBinary",
        "InventoryDevicePnp",
        "InventoryApplication",
        "InventoryApplicationShortcut",
    ]

    types = {
        Registry.RegSZ: "TEXT",
        Registry.RegExpandSZ: "TEXT",
        Registry.RegMultiSZ: "TEXT",
        Registry.RegDWord: "INTEGER",
        Registry.RegQWord: "INTEGER",
    }

    # conn = sqlite3.connect("amcache.sqlite")
    # cursor = conn.cursor()

    columns = {}
    for key_name in keys:
        key = reg.open(f"Root\\{key_name}")
        columns[key_name] = {}
        for subkey in key.subkeys():
            for value in subkey.values():
                if "default" in value.name():
                    continue
                column_name = value.name()
                column_type = types.get(value.value_type(), "TEXT")
                columns[key_name][column_name] = column_type

    # total_keys = 0
    # for key_name, cols in columns.items():
    #     print(f"{key_name}: {len(cols)} columns")
    #     total_keys += len(cols)

    # print(f"Total unique columns across all keys: {total_keys}")
        # table_statement = create_table_statement(key_name, [type('Column', (object,), {'name': k, 'data_type': v}) for k, v in columns.items()])
        # print(table_statement)
        # cursor.execute(table_statement)
        # conn.commit()

    # for key_name in keys:
    #     key = reg.open(f"Root\\{key_name}")
    #     for subkey in key.subkeys():
    #         column_names = []
    #         placeholders = []
    #         values = []
    #         for value in subkey.values():
    #             if "default" in value.name():
    #                 continue
    #             column_names.append(value.name())
    #             placeholders.append("?")
    #             values.append(str(value.value()))
    #         insert_statement = f'''
    #             INSERT INTO {key_name} ({", ".join(column_names)})
    #             VALUES ({", ".join(placeholders)})
    #         '''
    #         print(insert_statement, values)
    #         cursor.execute(insert_statement, values)
    #     conn.commit()
    # conn.close()

    import json
    print(json.dumps(columns, indent=4))
    def to_snake_case(text):
        import re
        # Replace any non-alphanumeric characters (except underscore) with spaces
        text = re.sub(r'[^a-zA-Z0-9_]', ' ', text)
        # Convert camelCase/PascalCase to snake_case by adding underscore before uppercase letters
        text = re.sub(r'(?<!^)(?=[A-Z])', '_', text)
        # Replace spaces with underscores and convert to lowercase
        return text.replace(' ', '_').lower()

    for key_name, cols in columns.items():
        print(f"type {key_name} struct {{")
        for col_name, col_type in cols.items():
            print(f"    {col_name} string `json:\"{to_snake_case(col_name)}\"`")
        print("}")
if __name__ == "__main__":
    main()