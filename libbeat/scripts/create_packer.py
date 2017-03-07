import os
import argparse

# Adds dev-tools/packer directory with the necessary files to a beat


def generate_packer(es_beats, abs_path, beat, beat_path, version):

    # create dev-tools/packer
    packer_path = abs_path + "/dev-tools/packer"

    print(packer_path)

    if os.path.isdir(packer_path):
        print("Dev tools already exists. Stopping...")
        return

    # create all directories needed
    os.makedirs(packer_path + "/beats")

    templates = es_beats + "/libbeat/scripts/dev-tools/packer"

    content = load_file(templates + "/version.yml", beat, beat_path, version)
    with open(packer_path + "/version.yml", "w") as f:
        f.write(content)

    content = load_file(templates + "/Makefile", beat, beat_path, version)
    with open(packer_path + "/Makefile", "w") as f:
        f.write(content)

    content = load_file(templates + "/config.yml", beat, beat_path, version)
    with open(packer_path + "/beats/" + beat + ".yml", "w") as f:
        f.write(content)

    print("Packer directories created")


def load_file(file, beat, beat_path, version):
    content = ""
    with open(file) as f:
        content = f.read()

    return content.replace("{beat}", beat).replace("{beat_path}", beat_path).replace("{version}", version)


if __name__ == "__main__":

    parser = argparse.ArgumentParser(description="Creates the beats packer structure")
    parser.add_argument("--beat", help="Beat name", default="test")
    parser.add_argument("--beat_path", help="Beat path", default="./")
    parser.add_argument("--es_beats", help="Beat path", default="../")
    parser.add_argument("--version", help="Beat version", default="0.1.0")

    args = parser.parse_args()

    # Fetches GOPATH and current execution directory. It is expected to run this script from the Makefile.
    gopath = os.environ['GOPATH'].split(os.pathsep)[0]
    # Normalise go path
    gopath = os.path.abspath(gopath)
    abs_path = os.path.abspath("./")

    # Removes the gopath + /src/ from the directory name to fetch the path
    beat_path = abs_path[len(gopath) + 5:]

    print(beat_path)
    print(abs_path)

    es_beats = os.path.abspath(args.es_beats)
    generate_packer(es_beats, abs_path, args.beat, beat_path, args.version)
