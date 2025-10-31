from regipy.registry import RegistryHive
import pathlib
import tabulate


def to_snake_case(text):
    # Handle empty string
    if not text:
        return text

    result = [text[0].lower()]
    # Iterate through remaining characters
    for i in range(1, len(text)):
        # Add underscore only when current char is upper and next char is lower
        # or when previous char is lower and current char is upper
        if (text[i].isupper() and 
            ((i + 1 < len(text) and text[i + 1].islower()) or 
             (text[i - 1].islower()))):
            result.extend(['_', text[i].lower()])
        else:
            result.append(text[i].lower())

    return ''.join(result)




hivepath = pathlib.Path(__file__).parent / "amcache.hve"
reg = RegistryHive(hivepath)
table_data = []
for key in reg.root.get_subkey("Root").iter_subkeys():
    snake_name = to_snake_case(key.name)
    subkey_count = len(list(key.iter_subkeys()))
    table_name = f"amcache_{snake_name.removeprefix('inventory_')}"
    included_in_extension = "Yes" if key.name in ["InventoryApplication", "InventoryApplicationFile", "InventoryApplicationShortcut", "InventoryDevicePnp", "InventoryDriverBinary"] else "No"
    table_data.append([key.name, subkey_count, included_in_extension, table_name if included_in_extension == "Yes" else "N/A"])

print(tabulate.tabulate(table_data, headers=["Key Name", "Record Count", "Included in Extension", "Table Name"], tablefmt="github"))
