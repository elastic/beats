package dtfmt

import (
	"errors"
	"time"
)

type prog struct {
	p []byte
}

const (
	opNone      byte = iota
	opCopy1          // copy next byte
	opCopy2          // copy next 2 bytes
	opCopy3          // copy next 3 bytes
	opCopy4          // copy next 4 bytes
	opCopyShort      // [op, len, content[len]]
	opCopyLong       // [op, len1, len, content[len1<<8 + len]]
	opNum            // [op, ft]
	opNumPadded      // [op, ft, digits]
	opTwoDigit       // [op, ft]
	opTextShort      // [op, ft]
	opTextLong       // [op, ft]
)

func (p prog) eval(bytes []byte, ctx *ctx, t time.Time) ([]byte, error) {
	for i := 0; i < len(p.p); {
		op := p.p[i]
		i++
		switch op {
		case opNone:

		case opCopy1:
			bytes = append(bytes, p.p[i])
			i++
		case opCopy2:
			bytes = append(bytes, p.p[i], p.p[i+1])
			i += 2
		case opCopy3:
			bytes = append(bytes, p.p[i], p.p[i+1], p.p[i+2])
			i += 3
		case opCopy4:
			bytes = append(bytes, p.p[i], p.p[i+1], p.p[i+2], p.p[i+3])
			i += 4
		case opCopyShort:
			l := int(p.p[i])
			i++
			bytes = append(bytes, p.p[i:i+l]...)
			i += l
		case opCopyLong:
			l := int(p.p[i])<<8 | int(p.p[i+1])
			i += 2
			bytes = append(bytes, p.p[i:i+l]...)
			i += l
		case opNum:
			ft := fieldType(p.p[i])
			i++
			v, err := getIntField(ft, ctx, t)
			if err != nil {
				return bytes, err
			}
			bytes = appendUnpadded(bytes, v)
		case opNumPadded:
			ft, digits := fieldType(p.p[i]), int(p.p[i+1])
			i += 2
			v, err := getIntField(ft, ctx, t)
			if err != nil {
				return bytes, err
			}
			bytes = appendPadded(bytes, v, digits)
		case opTwoDigit:
			ft := fieldType(p.p[i])
			i++
			v, err := getIntField(ft, ctx, t)
			if err != nil {
				return bytes, err
			}
			bytes = appendPadded(bytes, v%100, 2)
		case opTextShort:
			ft := fieldType(p.p[i])
			i++
			s, err := getTextFieldShort(ft, ctx, t)
			if err != nil {
				return bytes, err
			}
			bytes = append(bytes, s...)
		case opTextLong:
			ft := fieldType(p.p[i])
			i++
			s, err := getTextField(ft, ctx, t)
			if err != nil {
				return bytes, err
			}
			bytes = append(bytes, s...)
		default:
			return bytes, errors.New("unknown opcode")
		}
	}

	return bytes, nil
}

func makeProg(b ...byte) (prog, error) {
	return prog{b}, nil
}

func makeCopy(b []byte) (prog, error) {
	l := len(b)
	switch l {
	case 0:
		return prog{}, nil
	case 1:
		return makeProg(opCopy1, b[0])
	case 2:
		return makeProg(opCopy2, b[0], b[1])
	case 3:
		return makeProg(opCopy2, b[0], b[1], b[2])
	case 4:
		return makeProg(opCopy2, b[0], b[1], b[2], b[3])
	}

	if l < 256 {
		return prog{append([]byte{opCopyShort, byte(l)}, b...)}, nil
	}
	if l < (1 << 16) {
		l1 := byte(l >> 8)
		l2 := byte(l)
		return prog{append([]byte{opCopyLong, l1, l2}, b...)}, nil
	}

	return prog{}, errors.New("literal too long")
}
