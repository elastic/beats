import pickletools
v0 = [opcode for opcode in pickletools.opcodes if opcode.proto == 0]
v1 = [opcode for opcode in pickletools.opcodes if opcode.proto == 1] 
v2 = [opcode for opcode in pickletools.opcodes if opcode.proto == 2]
