package gotype

type symbolCache struct {
	m   map[string]*symbol
	lst symbolList
	max int
}

type symbol struct {
	value string
	prev  *symbol
	next  *symbol
}

type symbolList symbol

func (c *symbolCache) init(max int) {
	c.max = max
	c.m = make(map[string]*symbol, max)
	c.lst = symbolList{}
	c.lst.prev = (*symbol)(&c.lst)
	c.lst.next = (*symbol)(&c.lst)
}

func (c *symbolCache) enabled() bool {
	return c.m != nil
}

func (c *symbolCache) get(in []byte) string {
	if !c.enabled() {
		return string(in)
	}

	if sym := c.lookup(bytes2Str(in)); sym != nil {
		return sym.value
	}

	str := string(in)
	c.add(str)
	return str
}

func (c *symbolCache) lookup(value string) *symbol {
	sym := c.m[value]
	if sym != nil {
		removeLst(sym)
		c.lst.append(sym)
	}
	return sym
}

func (c *symbolCache) add(value string) {
	if len(c.m) == c.max {
		old := c.lst.pop()
		delete(c.m, old.value)
	}

	sym := &symbol{value: value}
	c.m[value] = sym
	c.lst.append(sym)
}

func (l *symbolList) empty() bool {
	s := (*symbol)(l)
	return s.next == s && s.prev == s
}

func (l *symbolList) append(sym *symbol) {
	head := (*symbol)(l)

	sym.prev = head.prev
	sym.next = head
	head.prev.next = sym
	head.prev = sym
}

func (l *symbolList) pop() (sym *symbol) {
	if !l.empty() {
		sym = l.next
		removeLst(sym)
	}
	return
}

func removeLst(s *symbol) {
	s.prev.next = s.next
	s.next.prev = s.prev
}
