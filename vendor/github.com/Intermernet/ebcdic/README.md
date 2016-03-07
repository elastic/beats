[![GoDoc](https://godoc.org/github.com/Intermernet/ebcdic?status.png)](https://godoc.org/github.com/Intermernet/ebcdic) [![Build Status](https://drone.io/github.com/Intermernet/ebcdic/status.png)](https://drone.io/github.com/Intermernet/ebcdic/latest) [![Coverage Status](https://coveralls.io/repos/Intermernet/ebcdic/badge.png?branch=master)](https://coveralls.io/r/Intermernet/ebcdic?branch=master)

EBCDIC / Unicode transcoding package for Go (Code page 37 only, `0x00` .. `0xFF`).

Example usage:

    package main
    
    import (
    	"fmt"
    
    	"github.com/Intermernet/ebcdic"
    )

    func main() {
    	input := []byte{0xc1, 0x95, 0x40, 0x81, 0x93, 0x93, 0x85, 0x87, 0x85, 0x84,
    		0x40, 0x83, 0x88, 0x81, 0x99, 0x81, 0x83, 0xa3, 0x85, 0x99, 0x40, 0xa2,
    		0x85, 0xa3, 0x40, 0xa4, 0xa2, 0x85, 0x84, 0x40, 0x96, 0x95, 0x40, 0xc9,
    		0xc2, 0xd4, 0x40, 0x84, 0x89, 0x95, 0x96, 0xa2, 0x81, 0xa4, 0x99, 0xa2,
    	}
    	fmt.Println(string(ebcdic.Decode(input)))
    }

will produce the Unicode string:

`An alleged character set used on IBM dinosaurs`

You can also convert Unicode text (Code point `00` .. `FF`) to EBCDIC.

See [godoc.org](https://godoc.org/github.com/Intermernet/ebcdic) for usage.

Conversion table data from http://www.unicode.org/Public/MAPPINGS/VENDORS/MICSFT/EBCDIC/CP037.TXT

Example conversion text from http://zvon.org/comp/r/ref-Jargon_file.html#Terms~EBCDIC