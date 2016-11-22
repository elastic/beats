package streambuf

// ASCII parsing support

import (
	"bytes"
	"errors"
)

var ErrExpectedDigit = errors.New("Expected digit")

// UntilCRLF collects all bytes until a CRLF ("\r\n") sequence is found. The
// returned byte slice will not contain the CRLF sequence.
// If CRLF was not found yet one of ErrNoMoreBytes or ErrUnexpectedEOB will be
// reported.
func (b *Buffer) UntilCRLF() ([]byte, error) {
	if b.err != nil {
		return nil, b.err
	}

	data := b.data[b.offset:]
	for i, byte := range data {
		if byte != '\r' {
			continue
		}

		if len(data) < i+2 {
			b.offset += i
			return nil, b.bufferEndError()
		}

		if data[i+1] != '\n' {
			// false alarm, continue
			continue
		}

		// yay, found
		end := i
		data = b.data[b.mark : b.offset+end]
		b.Advance(len(data) + 2)
		return data, nil
	}

	b.offset += len(data)
	return nil, b.bufferEndError()
}

// IgnoreSymbol will advance the read pointer until the first symbol not
// matching s is found.
func (b *Buffer) IgnoreSymbol(s uint8) error {
	if b.err != nil {
		return b.err
	}

	data := b.data[b.offset:]
	for i, byte := range data {
		if byte != s {
			b.Advance(b.offset + i - b.mark)
			return nil
		}
	}
	b.offset += len(data)
	return b.bufferEndError()
}

// IgnoreSymbols will advance the read pointer until the first symbol not matching
// set of symbols is found
func (b *Buffer) IgnoreSymbols(syms []byte) error {
	if b.err != nil {
		return b.err
	}

	data := b.data[b.offset:]
	for i, byte := range data {
		for _, other := range syms {
			if byte == other {
				goto next
			}
		}
		// no match
		b.Advance(b.offset + i - b.mark)
		return nil

	next:
	}
	b.offset += len(data)
	return b.bufferEndError()
}

// UntilSymbol collects all bytes until symbol s is found. If errOnEnd is set to
// true, the collected byte slice will be returned if no more bytes are available
// for parsing, but s has not matched yet.
func (b *Buffer) UntilSymbol(s uint8, errOnEnd bool) ([]byte, error) {
	if b.err != nil {
		return nil, b.err
	}

	data := b.data[b.offset:]
	for i, byte := range data {
		if byte == s {
			data := b.data[b.mark : b.offset+i]
			b.Advance(len(data))
			return data, nil
		}
	}

	if errOnEnd {
		b.offset += len(data)
		return nil, b.bufferEndError()
	}

	data = b.data[b.mark:]
	b.Advance(len(data))
	return data, nil
}

// UintASCII will parse unsigned number from Buffer.
func (b *Buffer) UintASCII(errOnEnd bool) (uint64, error) {
	if b.err != nil {
		return 0, b.err
	}
	if len(b.data) <= b.mark { // end of buffer
		return 0, b.bufferEndError()
	}

	end, err := b.asciiFindNumberEnd(b.offset, errOnEnd)
	if err != nil {
		return 0, err
	}

	// parse value
	value, err := doParseNumber(b.data[b.mark:end])
	if err != nil {
		return 0, err
	}
	b.Advance(end - b.mark)
	return value, nil
}

// IntASCII will parse (optionally) signed number from Buffer.
func (b *Buffer) IntASCII(errOnEnd bool) (int64, error) {
	if b.err != nil {
		return 0, b.err
	}
	if len(b.data) <= b.mark { // end of buffer
		return 0, b.bufferEndError()
	}

	// check signedness of number
	signed := b.data[b.mark] == '-'
	start := b.mark
	if signed {
		start++
		if len(b.data) <= start {
			return 0, b.bufferEndError()
		}
	} else if b.data[b.mark] == '+' {
		start++
		if len(b.data) <= start {
			return 0, b.bufferEndError()
		}
	}

	// adapt offset to point to start of number
	offset := b.offset
	if b.offset == b.mark {
		offset = start
	}

	end, err := b.asciiFindNumberEnd(offset, errOnEnd)
	if err != nil {
		return 0, err
	}

	value, err := doParseNumber(b.data[start:end])
	if err != nil {
		return 0, err
	}

	b.Advance(end - b.mark)
	if signed {
		return -int64(value), nil
	}
	return int64(value), nil
}

// MatchASCII checks the Buffer it's next byte sequence matched prefix. The
// read pointer is not advanced by AsciiPrefix.
func (b *Buffer) MatchASCII(prefix []byte) (bool, error) {
	if b.err != nil {
		return false, b.err
	}
	if !b.Avail(len(prefix)) {
		return false, b.bufferEndError()
	}

	has := bytes.HasPrefix(b.data[b.mark:], prefix)
	return has, nil
}

func (b *Buffer) asciiFindNumberEnd(start int, errOnEnd bool) (int, error) {
	// find end of number
	end := -1
	for i, byte := range b.data[start:] {
		if byte < '0' || '9' < byte {
			end = i + start
			break
		}
	}

	// check end
	if end < 0 {
		if errOnEnd {
			return -1, b.bufferEndError()
		}
		end = len(b.data)
	}

	return end, nil
}

func doParseNumber(buf []byte) (uint64, error) {
	if len(buf) == 0 {
		return 0, ErrExpectedDigit
	}

	var value uint64
	for _, byte := range buf {
		value = uint64(byte-'0') + 10*value
	}
	return value, nil
}

// binary parsing support
