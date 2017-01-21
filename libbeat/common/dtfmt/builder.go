package dtfmt

type builder struct {
	elements []element
}

func newBuilder() *builder {
	return &builder{}
}

func (b *builder) estimateSize() int {
	sz := 0
	for _, e := range b.elements {
		sz += e.estimateSize()
	}
	return sz
}

func (b *builder) createConfig() (ctxConfig, error) {
	cfg := ctxConfig{}
	for _, e := range b.elements {
		if err := e.requires(&cfg); err != nil {
			return ctxConfig{}, err
		}
	}
	return cfg, nil
}

func (b *builder) compile() (prog, error) {
	p := prog{}

	for _, e := range b.elements {
		tmp, err := e.compile()
		if err != nil {
			return prog{}, err
		}

		p.p = append(p.p, tmp.p...)
	}
	return p, nil
}

func (b *builder) optimize() {
	if len(b.elements) == 0 {
		return
	}

	// combine rune/string literals
	el := b.elements[:1]
	for _, e := range b.elements[1:] {
		last := el[len(el)-1]
		if r, ok := e.(runeLiteral); ok {
			if l, ok := last.(runeLiteral); ok {
				el[len(el)-1] = stringLiteral{
					append(append([]byte{}, string(l.r)...), string(r.r)...),
				}
			} else if l, ok := last.(stringLiteral); ok {
				el[len(el)-1] = stringLiteral{append(l.s, string(r.r)...)}
			} else {
				el = append(el, e)
			}
		} else if s, ok := e.(stringLiteral); ok {
			if l, ok := last.(runeLiteral); ok {
				el[len(el)-1] = stringLiteral{
					append(append([]byte{}, string(l.r)...), s.s...),
				}
			} else if l, ok := last.(stringLiteral); ok {
				el[len(el)-1] = stringLiteral{append(l.s, s.s...)}
			} else {
				el = append(el, e)
			}
		} else {
			el = append(el, e)
		}
	}
	b.elements = el
}

func (b *builder) add(e element) {
	b.elements = append(b.elements, e)
}

func (b *builder) millisOfSecond(digits int) {
	b.appendDecimal(ftMillisOfSecond, digits, 3)
}

func (b *builder) millisOfDay(digits int) {
	b.appendDecimal(ftMillisOfDay, digits, 8)
}

func (b *builder) secondOfMinute(digits int) {
	b.appendDecimal(ftSecondOfMinute, digits, 2)
}

func (b *builder) secondOfDay(digits int) {
	b.appendDecimal(ftSecondOfDay, digits, 5)
}

func (b *builder) minuteOfHour(digits int) {
	b.appendDecimal(ftMinuteOfHour, digits, 2)
}

func (b *builder) minuteOfDay(digits int) {
	b.appendDecimal(ftMinuteOfDay, digits, 4)
}

func (b *builder) hourOfDay(digits int) {
	b.appendDecimal(ftHourOfDay, digits, 2)
}

func (b *builder) clockhourOfDay(digits int) {
	b.appendDecimal(ftClockhourOfDay, digits, 2)
}

func (b *builder) hourOfHalfday(digits int) {
	b.appendDecimal(ftHourOfHalfday, digits, 2)
}

func (b *builder) clockhourOfHalfday(digits int) {
	b.appendDecimal(ftClockhourOfHalfday, digits, 2)
}

func (b *builder) dayOfWeek(digits int) {
	b.appendDecimal(ftDayOfWeek, digits, 1)
}

func (b *builder) dayOfMonth(digits int) {
	b.appendDecimal(ftDayOfMonth, digits, 2)
}

func (b *builder) dayOfYear(digits int) {
	b.appendDecimal(ftDayOfYear, digits, 3)
}

func (b *builder) weekOfWeekyear(digits int) {
	b.appendDecimal(ftWeekOfWeekyear, digits, 2)
}

func (b *builder) weekyear(minDigits, maxDigits int) {
	b.appendDecimal(ftWeekyear, minDigits, maxDigits)
}

func (b *builder) monthOfYear(digits int) {
	b.appendDecimal(ftMonthOfYear, digits, 2)
}

func (b *builder) year(minDigits, maxDigits int) {
	b.appendSigned(ftYear, minDigits, maxDigits)
}

func (b *builder) twoDigitYear() {
	b.add(twoDigitYear{ftYear})
}

func (b *builder) twoDigitWeekYear() {
	b.add(twoDigitYear{ftWeekyear})
}

func (b *builder) halfdayOfDayText() {
	b.appendText(ftHalfdayOfDay)
}

func (b *builder) dayOfWeekText() {
	b.appendText(ftDayOfWeek)
}

func (b *builder) dayOfWeekShortText() {
	b.appendShortText(ftDayOfWeek)
}

func (b *builder) monthOfYearText() {
	b.appendText(ftMonthOfYear)
}

func (b *builder) monthOfYearShortText() {
	b.appendShortText(ftMonthOfYear)
}

// TODO: add timezone support

func (b *builder) appendRune(r rune) {
	b.add(runeLiteral{r})
}

func (b *builder) appendLiteral(l string) {
	switch len(l) {
	case 0:
	case 1:
		b.add(runeLiteral{rune(l[0])})
	default:
		b.add(stringLiteral{[]byte(l)})
	}
}

func (b *builder) appendDecimalValue(ft fieldType, minDigits, maxDigits int, signed bool) {
	if maxDigits < minDigits {
		maxDigits = minDigits
	}

	if minDigits <= 1 {
		b.add(unpaddedNumber{ft, maxDigits, signed})
	} else {
		b.add(paddedNumber{ft, minDigits, maxDigits, signed})
	}
}

func (b *builder) appendDecimal(ft fieldType, minDigits, maxDigits int) {
	b.appendDecimalValue(ft, minDigits, maxDigits, false)
}

func (b *builder) appendSigned(ft fieldType, minDigits, maxDigits int) {
	b.appendDecimalValue(ft, minDigits, maxDigits, true)
}

func (b *builder) appendText(ft fieldType) {
	b.add(textField{ft, false})
}

func (b *builder) appendShortText(ft fieldType) {
	b.add(textField{ft, true})
}
