import os
import argparse

# Creates a new beat based on the given parameters

project_name = ""
github_name = ""
beat = ""
beat_path = ""
full_name = ""

def generate_beat(template_path, args):

    global project_name, github_name, beat, beat_path, full_name

    if args.project_name is not None:
        project_name = args.project_name

    if args.github_name is not None:
        github_name = args.github_name

    if args.beat_path is not None:
        beat_path = args.beat_path

    if args.full_name is not None:
        full_name = args.full_name

    read_input()
    process_file(template_path)

def read_input():
    """Requests input form the command line for empty variables if needed.
    """
    global project_name, github_name, beat, beat_path, full_name

    if project_name == "":
        project_name = raw_input("Beat Name [Examplebeat]: ") or "examplebeat"

    if github_name == "":
        github_name = raw_input("Your Github Name [your-github-name]: ") or "your-github-name"
    beat = project_name.lower()

    if beat_path == "":
        beat_path = raw_input("Beat Path [github.com/" + github_name + "/" + beat + "]: ") or "github.com/" + github_name + "/" + beat

    if full_name == "":
        full_name = raw_input("Firstname Lastname: ") or "Firstname Lastname"

def process_file(template_path):

    # Load path information
    generator_path = os.path.dirname(os.path.realpath(__file__))
    go_path = os.environ['GOPATH']

    for root, dirs, files in os.walk(generator_path + '/' + template_path + '/{beat}'):

        for file in files:

            full_path = root + "/" + file

            ## load file
            content = ""
            with open(full_path) as f:
                content = f.read()

            # process content
            content = replace_variables(content)

            # Write new path
            new_path = replace_variables(full_path).replace(".go.tmpl", ".go")

            # remove generator info and beat name from path
            file_path = new_path.replace(generator_path + "/" + template_path + "/" + beat, "")

            # New file path to write file content to
            write_file = go_path + "/src/" + beat_path + "/" + file_path

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
    return content.replace("{project_name}", project_name) \
        .replace("{github_name}", github_name) \
        .replace("{beat}", beat) \
        .replace("{Beat}", beat.capitalize()) \
        .replace("{beat_path}", beat_path) \
        .replace("{full_name}", full_name)


def get_parser():
    """Creates parser to parse script params
    """
    parser = argparse.ArgumentParser(description="Creates a beat")
    parser.add_argument("--project_name", help="Project name")
    parser.add_argument("--github_name", help="Github name")
    parser.add_argument("--beat_path", help="Beat path")
    parser.add_argument("--full_name", help="Full name")

    return parser

