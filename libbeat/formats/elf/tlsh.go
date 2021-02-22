package elf

import (
	"math"
	"sort"
	"strings"
)

var vTable = []int{
	1, 87, 49, 12, 176, 178, 102, 166, 121, 193, 6, 84, 249, 230, 44, 163,
	14, 197, 213, 181, 161, 85, 218, 80, 64, 239, 24, 226, 236, 142, 38, 200,
	110, 177, 104, 103, 141, 253, 255, 50, 77, 101, 81, 18, 45, 96, 31, 222,
	25, 107, 190, 70, 86, 237, 240, 34, 72, 242, 20, 214, 244, 227, 149, 235,
	97, 234, 57, 22, 60, 250, 82, 175, 208, 5, 127, 199, 111, 62, 135, 248,
	174, 169, 211, 58, 66, 154, 106, 195, 245, 171, 17, 187, 182, 179, 0, 243,
	132, 56, 148, 75, 128, 133, 158, 100, 130, 126, 91, 13, 153, 246, 216, 219,
	119, 68, 223, 78, 83, 88, 201, 99, 122, 11, 92, 32, 136, 114, 52, 10,
	138, 30, 48, 183, 156, 35, 61, 26, 143, 74, 251, 94, 129, 162, 63, 152,
	170, 7, 115, 167, 241, 206, 3, 150, 55, 59, 151, 220, 90, 53, 23, 131,
	125, 173, 15, 238, 79, 95, 89, 16, 105, 137, 225, 224, 217, 160, 37, 123,
	118, 73, 2, 157, 46, 116, 9, 145, 134, 228, 207, 212, 202, 215, 69, 229,
	27, 188, 67, 124, 168, 252, 42, 4, 29, 108, 21, 247, 19, 205, 39, 203,
	233, 40, 186, 147, 198, 192, 155, 33, 164, 191, 98, 204, 165, 180, 117, 76,
	140, 36, 210, 172, 41, 54, 159, 8, 185, 232, 113, 196, 231, 47, 146, 120,
	51, 65, 28, 144, 254, 221, 93, 189, 194, 139, 112, 43, 71, 109, 184, 209,
}

func bucketMapping(salt, i, j, k int) int {
	h := vTable[salt]
	h = vTable[h^i]
	h = vTable[h^j]
	h = vTable[h^k]
	return h
}

const (
	log1_5 = 0.4054651
	log1_3 = 0.26236426
	log1_1 = 0.095310180
)

// compute length portion of tlsh
func capturing(length int) int {
	var i int
	switch {
	case length <= 656:
		i = int(math.Floor(math.Log(float64(length)) / log1_5))
	case length <= 3199:
		i = int(math.Floor(math.Log(float64(length))/log1_3 - 8.72777))
	default:
		i = int(math.Floor(math.Log(float64(length))/log1_1 - 62.5472))
	}
	return i & 0xFF
}

const slidingWindowSize = 5
const buckets = 256

type tlshState struct {
	checksum       int
	checksumArray  []int
	checksumLength int
	bucket         []int64
	bucketCount    int
	window         []int
	dataLen        int
	codeSize       int
}

func newTlsh() *tlshState {
	bucketCount := 128
	checksumLength := 1

	return &tlshState{
		bucketCount:    bucketCount,
		checksumLength: checksumLength,
		codeSize:       bucketCount >> 2,
		window:         make([]int, slidingWindowSize),
		bucket:         make([]int64, buckets),
	}
}

func (t *tlshState) update(data []byte) {
	// Indexes into the sliding window. They cycle like
	// 0 4 3 2 1
	// 1 0 4 3 2
	// 2 1 0 4 3
	// 3 2 1 0 4
	// 4 3 2 1 0
	// 0 4 3 2 1
	// and so on
	j := t.dataLen % slidingWindowSize
	j1 := (j - 1 + slidingWindowSize) % slidingWindowSize
	j2 := (j - 2 + slidingWindowSize) % slidingWindowSize
	j3 := (j - 3 + slidingWindowSize) % slidingWindowSize
	j4 := (j - 4 + slidingWindowSize) % slidingWindowSize

	fedLen := t.dataLen
	for i := 0; i < len(data); i++ {
		t.window[j] = int(data[i])
		if fedLen >= 4 {
			// only calculate when input >= 5 bytes
			t.checksum = bucketMapping(0, t.window[j], t.window[j1], t.checksum)
			if t.checksumLength > 1 {
				t.checksumArray[0] = t.checksum
				for k := 1; k < t.checksumLength; k++ {
					// use calculated 1 byte checksums to expand the total checksum to 3 bytes
					t.checksumArray[k] = bucketMapping(t.checksumArray[k-1], t.window[j], t.window[j1], t.checksumArray[k])
				}
			}

			r := bucketMapping(2, t.window[j], t.window[j1], t.window[j2])
			t.bucket[r]++
			r = bucketMapping(3, t.window[j], t.window[j1], t.window[j3])
			t.bucket[r]++
			r = bucketMapping(5, t.window[j], t.window[j2], t.window[j3])
			t.bucket[r]++
			r = bucketMapping(7, t.window[j], t.window[j2], t.window[j4])
			t.bucket[r]++
			r = bucketMapping(11, t.window[j], t.window[j1], t.window[j4])
			t.bucket[r]++
			r = bucketMapping(13, t.window[j], t.window[j3], t.window[j4])
			t.bucket[r]++
		}
		// rotate the sliding window indexes
		j4, j3, j2, j1, j = j3, j2, j1, j, j4

		fedLen++
	}
	t.dataLen += len(data)
}

func median(data []int64) int64 {
	length := len(data)
	if length%2 != 0 {
		return data[length/2]
	}
	return data[length/2-1]
}

func (t *tlshState) findQuartile() []int64 {
	bucketCopy := make([]int64, t.bucketCount)
	copy(bucketCopy, t.bucket)
	sort.Slice(bucketCopy, func(i, j int) bool {
		return bucketCopy[i] < bucketCopy[j]
	})

	length := len(bucketCopy)
	// Find the cutoff places depeding on if
	// the input slice length is even or odd
	var c1 int
	var c2 int
	if length%2 == 0 {
		c1 = length / 2
		c2 = length / 2
	} else {
		c1 = (length - 1) / 2
		c2 = c1 + 1
	}

	return []int64{
		median(bucketCopy[:c1]),
		median(bucketCopy),
		median(bucketCopy[c2:]),
	}
}

func (t *tlshState) hash() string {
	if t.dataLen == 0 {
		return ""
	}
	quartiles := t.findQuartile()
	q1 := quartiles[0]
	q2 := quartiles[1]
	q3 := quartiles[2]

	code := make([]int, t.codeSize)
	for i := 0; i < t.codeSize; i++ {
		h := 0
		for j := 0; j < 4; j++ {
			k := t.bucket[4*i+j]
			if q3 < k {
				h += 3 << (j * 2)
			} else if q2 < k {
				h += 2 << (j * 2)
			} else if q1 < k {
				h += 1 << (j * 2)
			}
		}
		code[i] = h
	}

	lValue := capturing(t.dataLen)
	q1Ratio := int(float64(q1*100.0)/float64(q3)) & 0xF
	q2Ratio := int(float64(q2*100.0)/float64(q3)) & 0xF

	if t.checksumLength == 1 {
		return encode([]int{t.checksum}, lValue, q1Ratio, q2Ratio, code)
	}
	return encode(t.checksumArray, lValue, q1Ratio, q2Ratio, code)
}

var hexChars = []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'A', 'B', 'C', 'D', 'E', 'F'}

func writeHex(src int, builder *strings.Builder) {
	builder.WriteByte(hexChars[(src>>4)&0xF])
	builder.WriteByte(hexChars[src&0xF])
}

func writeHexSwapped(src int, builder *strings.Builder) {
	builder.WriteByte(hexChars[src&0xF])
	builder.WriteByte(hexChars[(src>>4)&0xF])
}

func encode(checksum []int, lValue, q1Ratio, q2Ratio int, codes []int) string {
	// extra 4 characters come from length and Q1 and Q2 ratio.
	hashStringLength := len(codes)*2 + len(checksum)*2 + 4
	var builder strings.Builder
	builder.Grow(hashStringLength)
	for k := 0; k < len(checksum); k++ {
		writeHexSwapped(checksum[k], &builder)
	}
	writeHexSwapped(lValue, &builder)
	writeHex(q1Ratio<<4|q2Ratio, &builder)
	for i := 0; i < len(codes); i++ {
		writeHex(codes[len(codes)-1-i], &builder)
	}
	return builder.String()
}
