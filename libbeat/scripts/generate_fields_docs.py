import yaml

def document_fields(output, section, SECTIONS, level=3):

    if "anchor" in section:
        output.write("[[exported-fields-{}]]\n".format(section["anchor"]))
    output.write("{} {} Fields\n\n".format(level*'=', section["name"]))

    if "description" in section:
        output.write("{}\n\n".format(section["description"]))

    if "fields" not in section or not section["fields"]:
        return

    output.write("\n")
    for field in section["fields"]:

        if "type" in field and field["type"] == "group":
            for sec, name in SECTIONS:
                if sec == field["name"]:
                    field["anchor"] = field["name"]
                    field["name"] = name
                    break
            document_fields(output, field, SECTIONS, level + 1)
        else:
            document_field(output, field, level + 1)


def document_field(output, field, level):

    if "path" not in field:
        field["path"] = field["name"]

    output.write("{} {}\n\n".format(level*'=', field["path"]))

    if "type" in field:
        output.write("type: {}\n\n".format(field["type"]))
    if "example" in field:
        output.write("example: {}\n\n".format(field["example"]))
    if "format" in field:
        output.write("format: {}\n\n".format(field["format"]))
    if "required" in field:
        output.write("required: {}\n\n".format(field["required"]))

    if "description" in field:
        output.write("{}\n\n".format(field["description"]))


def fields_to_asciidoc(input, output, SECTIONS, beat):

    dict = {'beat': beat}

    output.write("""
////
This file is generated! See etc/fields.yml and scripts/generate_field_docs.py
////

[[exported-fields]]
== Exported Fields

This document describes the fields that are exported by {beat}. They are
grouped in the following categories:

""".format(**dict))

    for doc, _ in SECTIONS:
        output.write("* <<exported-fields-{}>>\n".format(doc))
    output.write("\n")

    docs = yaml.load(input)

    for doc, name in SECTIONS:
        if doc in docs:
            section = docs[doc]
            if "type" in section:
                if section["type"] == "group":
                    section["name"] = name
                    section["anchor"] = doc
                    document_fields(output, section, SECTIONS)






if __name__ == "__main__":
    if len(sys.argv) != 3:
        print "Usage: %s file.yml file.asciidoc" % sys.argv[0]
        sys.exit(1)

    input = open(sys.argv[1], 'r')
    output = open(sys.argv[2], 'w')

    try:
        fields_to_asciidoc(input, output)
    finally:
        input.close()
        output.close()
