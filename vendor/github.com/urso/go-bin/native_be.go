// +build ppc64 mips64 mips s390x

package bin

import "encoding/binary"

// Architecture native encoding
var NativeEndian = binary.BigEndian

type I8 = I8be
type I16 = I16be
type I32 = I32be
type I64 = I64be

type U8 = U8be
type U16 = U16be
type U32 = U32be
type U64 = U64be
