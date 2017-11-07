package dtfmt

import (
	"math"
	"strconv"
)

func appendUnpadded(bs []byte, i int) []byte {
	return strconv.AppendInt(bs, int64(i), 10)
}

func appendPadded(bs []byte, i, sz int) []byte {
	if i < 0 {
		bs = append(bs, '-')
		i = -i
	}

	if i < 10 {
		for ; sz > 1; sz-- {
			bs = append(bs, '0')
		}
		return append(bs, byte(i)+'0')
	}
	if i < 100 {
		for ; sz > 2; sz-- {
			bs = append(bs, '0')
		}
		return strconv.AppendInt(bs, int64(i), 10)
	}

	digits := 0
	if i < 1000 {
		digits = 3
	} else if i < 10000 {
		digits = 4
	} else {
		digits = int(math.Log10(float64(i))) + 1
	}
	for ; sz > digits; sz-- {
		bs = append(bs, '0')
	}

	return strconv.AppendInt(bs, int64(i), 10)
}
