package plist

import (
	"encoding/base64"
	"encoding/xml"
	"io"
	"math"
	"time"
)

const xmlDOCTYPE = `<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
`

type xmlPlistGenerator struct {
	writer     io.Writer
	xmlEncoder *xml.Encoder
}

func (p *xmlPlistGenerator) generateDocument(root cfValue) {
	io.WriteString(p.writer, xml.Header)
	io.WriteString(p.writer, xmlDOCTYPE)

	plistStartElement := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: "plist",
		},
		Attr: []xml.Attr{{
			Name: xml.Name{
				Space: "",
				Local: "version"},
			Value: "1.0"},
		},
	}

	p.xmlEncoder.EncodeToken(plistStartElement)

	p.writePlistValue(root)

	p.xmlEncoder.EncodeToken(plistStartElement.End())
	p.xmlEncoder.Flush()
}

func (p *xmlPlistGenerator) writeDictionary(dict *cfDictionary) {
	dict.sort()
	startElement := xml.StartElement{Name: xml.Name{Local: "dict"}}
	p.xmlEncoder.EncodeToken(startElement)
	for i, k := range dict.keys {
		p.xmlEncoder.EncodeElement(k, xml.StartElement{Name: xml.Name{Local: "key"}})
		p.writePlistValue(dict.values[i])
	}
	p.xmlEncoder.EncodeToken(startElement.End())
}

func (p *xmlPlistGenerator) writeArray(a *cfArray) {
	startElement := xml.StartElement{Name: xml.Name{Local: "array"}}
	p.xmlEncoder.EncodeToken(startElement)
	for _, v := range a.values {
		p.writePlistValue(v)
	}
	p.xmlEncoder.EncodeToken(startElement.End())
}

func (p *xmlPlistGenerator) writePlistValue(pval cfValue) {
	if pval == nil {
		return
	}

	defer p.xmlEncoder.Flush()

	if dict, ok := pval.(*cfDictionary); ok {
		p.writeDictionary(dict)
		return
	} else if a, ok := pval.(*cfArray); ok {
		p.writeArray(a)
		return
	} else if uid, ok := pval.(cfUID); ok {
		p.writeDictionary(&cfDictionary{
			keys: []string{"CF$UID"},
			values: []cfValue{
				&cfNumber{
					signed: false,
					value:  uint64(uid),
				},
			},
		})
		return
	}

	// Everything here and beyond is encoded the same way: <key>value</key>
	key := ""
	var encodedValue interface{} = pval

	switch pval := pval.(type) {
	case cfString:
		key = "string"
	case *cfNumber:
		key = "integer"
		if pval.signed {
			encodedValue = int64(pval.value)
		} else {
			encodedValue = pval.value
		}
	case *cfReal:
		key = "real"
		encodedValue = pval.value
		switch {
		case math.IsInf(pval.value, 1):
			encodedValue = "inf"
		case math.IsInf(pval.value, -1):
			encodedValue = "-inf"
		case math.IsNaN(pval.value):
			encodedValue = "nan"
		}
	case cfBoolean:
		key = "false"
		b := bool(pval)
		if b {
			key = "true"
		}
		encodedValue = ""
	case cfData:
		key = "data"
		encodedValue = xml.CharData(base64.StdEncoding.EncodeToString([]byte(pval)))
	case cfDate:
		key = "date"
		encodedValue = time.Time(pval).In(time.UTC).Format(time.RFC3339)
	}

	if key != "" {
		err := p.xmlEncoder.EncodeElement(encodedValue, xml.StartElement{Name: xml.Name{Local: key}})
		if err != nil {
			panic(err)
		}
	}
}

func (p *xmlPlistGenerator) Indent(i string) {
	p.xmlEncoder.Indent("", i)
}

func newXMLPlistGenerator(w io.Writer) *xmlPlistGenerator {
	mw := mustWriter{w}
	return &xmlPlistGenerator{mw, xml.NewEncoder(mw)}
}
