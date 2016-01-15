package flows

import "sync"

type Var interface{}

type Int struct {
	i int
}

type Float struct {
	i int
}

type counterReg struct {
	mutex sync.Mutex

	ints   counterTypeReg
	floats counterTypeReg
}

type counterTypeReg struct {
	names []string
}

type flowStats struct {
	ints   []int64
	floats []float64
}

func (c *Int) Add(f *Flow, delta int64) {
	ints := f.stats.ints
	if c.i < len(ints) {
		ints[c.i] += delta
	}
}

func (c *Int) Set(f *Flow, value int64) {
	ints := f.stats.ints
	if c.i < len(ints) {
		ints[c.i] = value
	}
}

func (c *Float) Add(f *Flow, delta float64) {
	floats := f.stats.floats
	if c.i < len(floats) {
		floats[c.i] += delta
	}
}

func (c *Float) Set(f *Flow, value float64) {
	floats := f.stats.floats
	if c.i < len(floats) {
		floats[c.i] = value
	}
}

func (c *counterReg) newInt(name string) (*Int, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	i, err := c.ints.reg(name)
	if err != nil {
		return nil, err
	}
	return &Int{i}, nil
}

func (c *counterReg) newFloat(name string) (*Float, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	i, err := c.floats.reg(name)
	if err != nil {
		return nil, err
	}
	return &Float{i}, nil
}

// XXX:
//  - error on index > int max
//  - error if already in use
func (reg *counterTypeReg) reg(name string) (int, error) {
	i := len(reg.names)
	reg.names = append(reg.names, name)
	return i, nil
}

func (reg *counterTypeReg) getNames() []string {
	return reg.names
}

func newFlowStats(reg *counterReg) *flowStats {
	s := &flowStats{}
	s.init(reg)
	return s
}

func (s *flowStats) init(reg *counterReg) {
	reg.mutex.Lock()
	defer reg.mutex.Unlock()

	s.ints = make([]int64, len(reg.ints.names))
	s.floats = make([]float64, len(reg.floats.names))
}
