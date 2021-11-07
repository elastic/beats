package cap

import "errors"

// GetFlag determines if the requested Value is enabled in the
// specified Flag of the capability Set.
func (c *Set) GetFlag(vec Flag, val Value) (bool, error) {
	if c == nil || len(c.flat) == 0 {
		// Checked this first, because otherwise we are sure
		// cInit has been called.
		return false, ErrBadSet
	}
	offset, mask, err := bitOf(vec, val)
	if err != nil {
		return false, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.flat[offset][vec]&mask != 0, nil
}

// SetFlag sets the requested bits to the indicated enable state. This
// function does not perform any security checks, so values can be set
// out-of-order. Only when the Set is used to SetProc() etc., will the
// bits be checked for validity and permission by the kernel. If the
// function returns an error, the Set will not be modified.
func (c *Set) SetFlag(vec Flag, enable bool, val ...Value) error {
	if c == nil || len(c.flat) == 0 {
		// Checked this first, because otherwise we are sure
		// cInit has been called.
		return ErrBadSet
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// Make a backup.
	replace := make([]uint32, words)
	for i := range replace {
		replace[i] = c.flat[i][vec]
	}
	var err error
	for _, v := range val {
		offset, mask, err2 := bitOf(vec, v)
		if err2 != nil {
			err = err2
			break
		}
		if enable {
			c.flat[offset][vec] |= mask
		} else {
			c.flat[offset][vec] &= ^mask
		}
	}
	if err == nil {
		return nil
	}
	// Clean up.
	for i, bits := range replace {
		c.flat[i][vec] = bits
	}
	return err
}

// Clear fully clears a capability set.
func (c *Set) Clear() error {
	if c == nil || len(c.flat) == 0 {
		return ErrBadSet
	}
	// startUp.Do(cInit) is not called here because c cannot be
	// initialized except via this package and doing that will
	// perform that call at least once (sic).
	c.mu.Lock()
	defer c.mu.Unlock()
	c.flat = make([]data, words)
	c.nsRoot = 0
	return nil
}

// Fill copies the from flag values into the to flag. With this
// function, you can raise all of the permitted values in the
// effective flag with c.Fill(cap.Effective, cap.Permitted).
func (c *Set) Fill(to, from Flag) error {
	if c == nil || len(c.flat) == 0 {
		return ErrBadSet
	}
	if to > Inheritable || from > Inheritable {
		return ErrBadValue
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.flat {
		c.flat[i][to] = c.flat[i][from]
	}
	return nil
}

// ErrBadValue indicates a bad capability value was specified.
var ErrBadValue = errors.New("bad capability value")

// bitOf converts from a Value into the offset and mask for a specific
// Value bit in the compressed (kernel ABI) representation of a
// capabilities. If the requested bit is unsupported, an error is
// returned.
func bitOf(vec Flag, val Value) (uint, uint32, error) {
	if vec > Inheritable || val > Value(words*32) {
		return 0, 0, ErrBadValue
	}
	u := uint(val)
	return u / 32, uint32(1) << (u % 32), nil
}

// allMask returns the mask of valid bits in the all mask for index.
func allMask(index uint) (mask uint32) {
	if maxValues == 0 {
		panic("uninitialized package")
	}
	base := 32 * uint(index)
	if maxValues <= base {
		return
	}
	if maxValues >= 32+base {
		mask = ^mask
		return
	}
	mask = uint32((uint64(1) << (maxValues % 32)) - 1)
	return
}

// forceFlag sets 'all' capability values (supported by the kernel) of
// a specified Flag to enable.
func (c *Set) forceFlag(vec Flag, enable bool) error {
	if c == nil || len(c.flat) == 0 || vec > Inheritable {
		return ErrBadSet
	}
	m := uint32(0)
	if enable {
		m = ^m
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.flat {
		c.flat[i][vec] = m & allMask(uint(i))
	}
	return nil
}

// ClearFlag clears all the Values associated with the specified Flag.
func (c *Set) ClearFlag(vec Flag) error {
	return c.forceFlag(vec, false)
}

// Compare returns 0 if c and d are identical in content. Otherwise,
// this function returns a non-zero value of 3 independent bits:
// (differE ? 1:0) | (differP ? 2:0) | (differI ? 4:0). The Differs()
// function can be used to test for a difference in a specific Flag.
//
// This function is deprecated in favor of (*Set).Cf().
func (c *Set) Compare(d *Set) (uint, error) {
	u, err := c.Cf(d)
	return uint(u), err
}

// Cf returns 0 if c and d are identical. A non-zero Diff value
// captures a simple macroscopic summary of how they differ. The
// (Diff).Has() function can be used to determine how the two
// capability sets differ.
func (c *Set) Cf(d *Set) (Diff, error) {
	if c == nil || len(c.flat) == 0 || d == nil || len(d.flat) == 0 {
		return 0, ErrBadSet
	}
	var cf Diff
	for i := 0; i < words; i++ {
		if c.flat[i][Effective]^d.flat[i][Effective] != 0 {
			cf |= effectiveDiff
		}
		if c.flat[i][Permitted]^d.flat[i][Permitted] != 0 {
			cf |= permittedDiff
		}
		if c.flat[i][Inheritable]^d.flat[i][Inheritable] != 0 {
			cf |= inheritableDiff
		}
	}
	return cf, nil
}

// Differs processes the result of Compare and determines if the
// Flag's components were different.
//
// Use of this function is deprecated in favor of the (Diff).Has()
// function, where Diff is returned as a result of the (*Set).Cf()
// function.
func Differs(cf uint, vec Flag) bool {
	return cf&(1<<vec) != 0
}

// Has processes the Diff result of (*Set).Cf() and determines if the
// Flag's components were different in that result.
func (cf Diff) Has(vec Flag) bool {
	return uint(cf)&(1<<vec) != 0
}
