import os
import argparse

# Creates a tcp protocol

protocol = ""
plugin_type = ""
plugin_var = ""


def generate_protocol():
    read_input()
    process_file()


def read_input():
    """Requests input form the command line for empty variables if needed.
    """
    global protocol, plugin_type, plugin_var

    if protocol == "":
        protocol = raw_input("Protocol Name [exampletcp]: ") or "exampletcp"

    protocol = protocol.lower()

    plugin_type = protocol + "Plugin"
    plugin_var = protocol[0] + "p"


def process_file():

    # Load path information
    generator_path = os.path.dirname(os.path.realpath(__file__))
    go_path = os.environ['GOPATH']

    for root, dirs, files in os.walk(generator_path + '/tcp-protocol/{protocol}'):

        for file in files:

            full_path = root + "/" + file

            # load file
            content = ""
            with open(full_path) as f:
                content = f.read()

            # process content
            content = replace_variables(content)

            # Write new path
            new_path = replace_variables(full_path).replace(".go.tmpl", ".go")

            # remove generator info from path
            file_path = new_path.replace(generator_path + "/tcp-protocol/", "")

            # New file path to write file content to
            write_file = "protos/" + file_path

            # Create parent directory if it does not exist yet
            dir = os.path.dirname(write_file)
            if not os.path.exists(dir):
                os.makedirs(dir)

            # Write file to new location
            with open(write_file, 'w') as f:
                f.write(content)


def replace_variables(content):
    """Replace all template variables with the actual values
    """
    return content.replace("{protocol}", protocol) \
        .replace("{plugin_var}", plugin_var) \
        .replace("{plugin_type}", plugin_type)


if __name__ == "__main__":

    parser = argparse.ArgumentParser(description="Creates a beat")
    parser.add_argument("--protocol", help="Protocol name")

    args = parser.parse_args()

    if args.protocol is not None:
        protocol = args.protocol

    generate_protocol()
