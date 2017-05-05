import pickletools
import sys
import textwrap
import versions

def print_opcode(opcode):
    sys.stdout.write("##Name: %s\n" % opcode.name)
    sys.stdout.write("* Opcode: 0x%X\n" % ord(opcode.code))
    sys.stdout.write("* Stack before: `%s`\n" % opcode.stack_before)
    sys.stdout.write("* Stack after: `%s`\n" % opcode.stack_after)
    sys.stdout.write("\n")
    sys.stdout.write("###Description\n")
    sys.stdout.write(opcode.doc)
    sys.stdout.write("\n")

sys.stdout.write("#Version 0\n")

for opcode in versions.v0:
    print_opcode(opcode)

sys.stdout.write("#Version 1\n")
for opcode in versions.v1:
    print_opcode(opcode)
    
sys.stdout.write("#Version 2\n")
for opcode in versions.v1:
    print_opcode(opcode)
