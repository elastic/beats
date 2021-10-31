// Copyright (c) 2012-2015 Ugorji Nwoke. All rights reserved.
// Use of this source code is governed by a MIT license found in the LICENSE file.

package codec

// All non-std package dependencies live in this file,
// so porting to different environment is easy (just update functions).

func pruneSignExt(v []byte, pos bool) (n int) {
	if len(v) < 2 {
	} else if pos && v[0] == 0 {
		for ; v[n] == 0 && n+1 < len(v) && (v[n+1]&(1<<7) == 0); n++ {
		}
	} else if !pos && v[0] == 0xff {
		for ; v[n] == 0xff && n+1 < len(v) && (v[n+1]&(1<<7) != 0); n++ {
		}
	}
	return
}

func halfFloatToFloatBits(h uint16) (f uint32) {
	// retrofitted from:
	// - OGRE (Object-Oriented Graphics Rendering Engine)
	//   function: halfToFloatI https://www.ogre3d.org/docs/api/1.9/_ogre_bitwise_8h_source.html

	s := uint32(h >> 15)
	m := uint32(h & 0x03ff)
	e := int32((h >> 10) & 0x1f)

	if e == 0 {
		if m == 0 { // plus or minus 0
			return s << 31
		}
		// Denormalized number -- renormalize it
		for (m & 0x0400) == 0 {
			m <<= 1
			e -= 1
		}
		e += 1
		m &= ^uint32(0x0400)
	} else if e == 31 {
		if m == 0 { // Inf
			return (s << 31) | 0x7f800000
		}
		return (s << 31) | 0x7f800000 | (m << 13) // NaN
	}
	e = e + (127 - 15)
	m = m << 13
	return (s << 31) | (uint32(e) << 23) | m
}

func floatToHalfFloatBits(i uint32) (h uint16) {
	// retrofitted from:
	// - OGRE (Object-Oriented Graphics Rendering Engine)
	//   function: halfToFloatI https://www.ogre3d.org/docs/api/1.9/_ogre_bitwise_8h_source.html
	// - http://www.java2s.com/example/java-utility-method/float-to/floattohalf-float-f-fae00.html
	s := (i >> 16) & 0x8000
	e := int32(((i >> 23) & 0xff) - (127 - 15))
	m := i & 0x7fffff

	var h32 uint32

	if e <= 0 {
		if e < -10 { // zero
			h32 = s // track -0 vs +0
		} else {
			m = (m | 0x800000) >> uint32(1-e)
			h32 = s | (m >> 13)
		}
	} else if e == 0xff-(127-15) {
		if m == 0 { // Inf
			h32 = s | 0x7c00
		} else { // NAN
			m >>= 13
			var me uint32
			if m == 0 {
				me = 1
			}
			h32 = s | 0x7c00 | m | me
		}
	} else {
		if e > 30 { // Overflow
			h32 = s | 0x7c00
		} else {
			h32 = s | (uint32(e) << 10) | (m >> 13)
		}
	}
	h = uint16(h32)
	return
}

// GrowCap will return a new capacity for a slice, given the following:
//   - oldCap: current capacity
//   - unit: in-memory size of an element
//   - num: number of elements to add
func growCap(oldCap, unit, num int) (newCap int) {
	// appendslice logic (if cap < 1024, *2, else *1.25):
	//   leads to many copy calls, especially when copying bytes.
	//   bytes.Buffer model (2*cap + n): much better for bytes.
	// smarter way is to take the byte-size of the appended element(type) into account

	// maintain 2 thresholds:
	// t1: if cap <= t1, newcap = 2x
	// t2: if cap <= t2, newcap = 1.5x
	//     else          newcap = 1.25x
	//
	// t1, t2 >= 1024 always.
	// This means that, if unit size >= 16, then always do 2x or 1.25x (ie t1, t2, t3 are all same)
	//
	// With this, appending for bytes increase by:
	//    100% up to 4K
	//     75% up to 16K
	//     25% beyond that

	// unit can be 0 e.g. for struct{}{}; handle that appropriately
	if unit <= 0 {
		if uint64(^uint(0)) == ^uint64(0) { // 64-bit
			var maxInt64 uint64 = 1<<63 - 1 // prevent failure with overflow int on 32-bit (386)
			return int(maxInt64)            // math.MaxInt64
		}
		return 1<<31 - 1 //  math.MaxInt32
	}

	// handle if num < 0, cap=0, etc.

	var t1, t2 int // thresholds
	if unit <= 4 {
		t1, t2 = 4*1024, 16*1024
	} else if unit <= 16 {
		t1, t2 = unit*1*1024, unit*4*1024
	} else {
		t1, t2 = 1024, 1024
	}

	if oldCap <= 0 {
		newCap = 2
	} else if oldCap <= t1 { // [0,t1]
		newCap = oldCap * 8 / 4
	} else if oldCap <= t2 { // (t1,t2]
		newCap = oldCap * 6 / 4
	} else { // (t2,infinity]
		newCap = oldCap * 5 / 4
	}

	if num > 0 && newCap < num+oldCap {
		newCap = num + oldCap
	}

	// ensure newCap takes multiples of a cache line (size is a multiple of 64)
	t1 = newCap * unit
	t2 = t1 % 64
	if t2 != 0 {
		t1 += 64 - t2
		newCap = t1 / unit
	}

	return
}
