// boolexp.g4
grammar Boolexp;

// Tokens
EQ: '==';
NEQ: '!=';
GT: '>';
LT: '<';
GTE: '>=';
LTE: '<=';
AND: 'and' | 'AND' | '&&';
OR: 'or' | 'OR' | '||';
TRUE: 'true' | 'TRUE';
FALSE: 'false' | 'FALSE';
FLOAT: [0-9]+ '.' [0-9]+;
NUMBER: [0-9]+;
WHITESPACE: [ \r\n\t]+ -> skip;
NOT: 'NOT' | '!' | 'not';
VARIABLE: BEGIN_VARIABLE [a-zA-Z0-9_.]+('.'[a-zZ0-9_]+)* END_VARIABLE;
METHODNAME: [a-zA-Z_] [a-zA-Z0-9_]*;
TEXT : '\'' ~[\r\n']* '\'';
LPAR: '(';
RPAR: ')';
fragment BEGIN_VARIABLE: '%{[';
fragment END_VARIABLE: ']}';

expList: exp EOF;

exp
: LPAR exp RPAR # ExpInParen
| NOT exp # ExpNot
| left=exp EQ right=exp # ExpArithmeticEQ
| left=exp NEQ right=exp # ExpArithmeticNEQ
| left=exp LTE right=exp # ExpArithmeticLTE
| left=exp GTE right=exp # ExpArithmeticGTE
| left=exp LT right=exp # ExpArithmeticLT
| left=exp GT right=exp # ExpArithmeticGT
| left=exp AND right=exp # ExpLogicalAnd
| left=exp OR right=exp # ExpLogicalOR
| boolean # ExpBoolean
| VARIABLE # ExpVariable
| METHODNAME LPAR arguments? RPAR # ExpFunction
| TEXT # ExpText
| FLOAT # ExpFloat
| NUMBER # ExpNumber
;

boolean
: TRUE | FALSE
;

arguments
: exp( ',' exp)*
;

