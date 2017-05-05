import versions
from io import open
import itertools

pkg_stmt = u'package stalecucumber\n\n'

def make_name(n):
	return "opcode_%s" % n

def write_opcode(fout,opcode):
	func_name = make_name(opcode.name)
	fout.write(u'/**\n')
	fout.write(u"Opcode: %s (0x%x)\n%s**\n" % (opcode.name,ord(opcode.code),opcode.doc))
	fout.write(u"Stack before: %s\n" % opcode.stack_before)
	fout.write(u"Stack after: %s\n" % opcode.stack_after)
	fout.write(u'**/\n')
	fout.write(u"func (pm *PickleMachine) %s () error {\n" % func_name)
	fout.write(u'return ErrOpcodeNotImplemented\n')
	fout.write(u'}\n\n')

with open('protocol_0.go','w') as fout:
	fout.write(pkg_stmt) 
	for opcode in versions.v0:
		write_opcode(fout,opcode)

with open('protocol_1.go','w') as fout:
	fout.write(pkg_stmt) 
	for opcode in versions.v1:
		write_opcode(fout,opcode)

with open('protocol_2.go','w') as fout:
	fout.write(pkg_stmt) 
	for opcode in versions.v2:
		write_opcode(fout,opcode)


all_opcodes = [versions.v0,versions.v1,versions.v2]

def opcode_to_const_name(o):
	return u'OPCODE_%s' % o.name

with open('opcodes.go','w') as fout:
	fout.write(pkg_stmt)
	
	for opcode in itertools.chain(*all_opcodes):
			fout.write(u'const ')
			fout.write(opcode_to_const_name(opcode))
			fout.write(u' = ')
			fout.write(u'0x%x' % ord(opcode.code))
			fout.write(u'\n')

with open('populate_jump_list.go','w') as fout:
	fout.write(pkg_stmt)

	fout.write(u'func populateJumpList(jl *opcodeJumpList) {\n')
	for opcode in itertools.chain(*all_opcodes):
		fout.write(u"\tjl[%s] = (*PickleMachine).%s\n" % (opcode_to_const_name(opcode),make_name(opcode.name)))
	fout.write(u"}\n\n")

		
