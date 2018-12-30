// +build amd64 386 arm arm64 ppc64le mips64le mipsle

package bin

import "encoding/binary"

// Architecture native encoding
var NativeEndian = binary.LittleEndian

type I8 = I8le
type I16 = I16le
type I32 = I32le
type I64 = I64le

type U8 = U8le
type U16 = U16le
type U32 = U32le
type U64 = U64le
