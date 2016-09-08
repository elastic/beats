package gocsv

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"reflect"
)

// Decoder .
type Decoder interface {
	getCSVRows() ([][]string, error)
}

// SimpleDecoder .
type SimpleDecoder interface {
	getCSVRow() ([]string, error)
}

type decoder struct {
	in         io.Reader
	csvDecoder *csvDecoder
}

func newDecoder(in io.Reader) *decoder {
	return &decoder{in: in}
}

func (decode *decoder) getCSVRows() ([][]string, error) {
	return getCSVReader(decode.in).ReadAll()
}

func (decode *decoder) getCSVRow() ([]string, error) {
	if decode.csvDecoder == nil {
		decode.csvDecoder = &csvDecoder{getCSVReader(decode.in)}
	}
	return decode.csvDecoder.Read()
}

type csvDecoder struct {
	*csv.Reader
}

func (c csvDecoder) getCSVRows() ([][]string, error) {
	return c.ReadAll()
}

func (c csvDecoder) getCSVRow() ([]string, error) {
	return c.Read()
}

func maybeMissingStructFields(structInfo []fieldInfo, headers []string) error {
	if len(structInfo) == 0 {
		return nil
	}

	headerMap := make(map[string]struct{}, len(headers))
	for idx := range headers {
		headerMap[headers[idx]] = struct{}{}
	}

	for _, info := range structInfo {
		found := false
		for _, key := range info.keys {
			if _, ok := headerMap[key]; ok {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("found unmatched struct field with tags %v", info.keys)
		}
	}
	return nil
}

// Check that no header name is repeated twice
func maybeDoubleHeaderNames(headers []string) error {
	headerMap := make(map[string]bool, len(headers))
	for _, v := range headers {
		if _, ok := headerMap[v]; ok {
			return fmt.Errorf("Repeated header name: %v", v)
		}
		headerMap[v] = true
	}
	return nil
}

func readTo(decoder Decoder, out interface{}) error {
	outValue, outType := getConcreteReflectValueAndType(out) // Get the concrete type (not pointer) (Slice<?> or Array<?>)
	if err := ensureOutType(outType); err != nil {
		return err
	}
	outInnerWasPointer, outInnerType := getConcreteContainerInnerType(outType) // Get the concrete inner type (not pointer) (Container<"?">)
	if err := ensureOutInnerType(outInnerType); err != nil {
		return err
	}
	csvRows, err := decoder.getCSVRows() // Get the CSV csvRows
	if err != nil {
		return err
	}
	if len(csvRows) == 0 {
		return errors.New("empty csv file given")
	}
	if err := ensureOutCapacity(&outValue, len(csvRows)); err != nil { // Ensure the container is big enough to hold the CSV content
		return err
	}
	outInnerStructInfo := getStructInfo(outInnerType) // Get the inner struct info to get CSV annotations
	if len(outInnerStructInfo.Fields) == 0 {
		return errors.New("no csv struct tags found")
	}

	headers := csvRows[0]
	body := csvRows[1:]

	csvHeadersLabels := make(map[int]*fieldInfo, len(outInnerStructInfo.Fields)) // Used to store the correspondance header <-> position in CSV

	for i, csvColumnHeader := range headers {
		if fieldInfo := getCSVFieldPosition(csvColumnHeader, outInnerStructInfo); fieldInfo != nil {
			csvHeadersLabels[i] = fieldInfo
		}
	}

	if FailIfUnmatchedStructTags {
		if err := maybeMissingStructFields(outInnerStructInfo.Fields, headers); err != nil {
			return err
		}
	}
	if FailIfDoubleHeaderNames {
		if err := maybeDoubleHeaderNames(headers); err != nil {
			return err
		}
	}

	for i, csvRow := range body {
		outInner := createNewOutInner(outInnerWasPointer, outInnerType)
		for j, csvColumnContent := range csvRow {
			if fieldInfo, ok := csvHeadersLabels[j]; ok { // Position found accordingly to header name
				if err := setInnerField(&outInner, outInnerWasPointer, fieldInfo.IndexChain, csvColumnContent); err != nil { // Set field of struct
					return &csv.ParseError{
						Line:   i + 2, //add 2 to account for the header & 0-indexing of arrays
						Column: j + 1,
						Err:    err,
					}
				}
			}
		}
		outValue.Index(i).Set(outInner)
	}
	return nil
}

func readEach(decoder SimpleDecoder, c interface{}) error {
	headers, err := decoder.getCSVRow()
	if err != nil {
		return err
	}
	outValue, outType := getConcreteReflectValueAndType(c) // Get the concrete type (not pointer) (Slice<?> or Array<?>)
	if err := ensureOutType(outType); err != nil {
		return err
	}
	defer outValue.Close()
	outInnerWasPointer, outInnerType := getConcreteContainerInnerType(outType) // Get the concrete inner type (not pointer) (Container<"?">)
	if err := ensureOutInnerType(outInnerType); err != nil {
		return err
	}
	outInnerStructInfo := getStructInfo(outInnerType) // Get the inner struct info to get CSV annotations
	if len(outInnerStructInfo.Fields) == 0 {
		return errors.New("no csv struct tags found")
	}
	csvHeadersLabels := make(map[int]*fieldInfo, len(outInnerStructInfo.Fields)) // Used to store the correspondance header <-> position in CSV
	for i, csvColumnHeader := range headers {
		if fieldInfo := getCSVFieldPosition(csvColumnHeader, outInnerStructInfo); fieldInfo != nil {
			csvHeadersLabels[i] = fieldInfo
		}
	}
	if err := maybeMissingStructFields(outInnerStructInfo.Fields, headers); err != nil {
		if FailIfUnmatchedStructTags {
			return err
		}
	}
	if FailIfDoubleHeaderNames {
		if err := maybeDoubleHeaderNames(headers); err != nil {
			return err
		}
	}
	i := 0
	for {
		line, err := decoder.getCSVRow()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		outInner := createNewOutInner(outInnerWasPointer, outInnerType)
		for j, csvColumnContent := range line {
			if fieldInfo, ok := csvHeadersLabels[j]; ok { // Position found accordingly to header name
				if err := setInnerField(&outInner, outInnerWasPointer, fieldInfo.IndexChain, csvColumnContent); err != nil { // Set field of struct
					return &csv.ParseError{
						Line:   i + 2, //add 2 to account for the header & 0-indexing of arrays
						Column: j + 1,
						Err:    err,
					}
				}
			}
		}
		outValue.Send(outInner)
		i++
	}
	return nil
}

// Check if the outType is an array or a slice
func ensureOutType(outType reflect.Type) error {
	switch outType.Kind() {
	case reflect.Slice:
		fallthrough
	case reflect.Chan:
		fallthrough
	case reflect.Array:
		return nil
	}
	return fmt.Errorf("cannot use " + outType.String() + ", only slice or array supported")
}

// Check if the outInnerType is of type struct
func ensureOutInnerType(outInnerType reflect.Type) error {
	switch outInnerType.Kind() {
	case reflect.Struct:
		return nil
	}
	return fmt.Errorf("cannot use " + outInnerType.String() + ", only struct supported")
}

func ensureOutCapacity(out *reflect.Value, csvLen int) error {
	switch out.Kind() {
	case reflect.Array:
		if out.Len() < csvLen-1 { // Array is not big enough to hold the CSV content (arrays are not addressable)
			return fmt.Errorf("array capacity problem: cannot store %d %s in %s", csvLen-1, out.Type().Elem().String(), out.Type().String())
		}
	case reflect.Slice:
		if !out.CanAddr() && out.Len() < csvLen-1 { // Slice is not big enough tho hold the CSV content and is not addressable
			return fmt.Errorf("slice capacity problem and is not addressable (did you forget &?)")
		} else if out.CanAddr() && out.Len() < csvLen-1 {
			out.Set(reflect.MakeSlice(out.Type(), csvLen-1, csvLen-1)) // Slice is not big enough, so grows it
		}
	}
	return nil
}

func getCSVFieldPosition(key string, structInfo *structInfo) *fieldInfo {
	for _, field := range structInfo.Fields {
		if field.matchesKey(key) {
			return &field
		}
	}
	return nil
}

func createNewOutInner(outInnerWasPointer bool, outInnerType reflect.Type) reflect.Value {
	if outInnerWasPointer {
		return reflect.New(outInnerType)
	}
	return reflect.New(outInnerType).Elem()
}

func setInnerField(outInner *reflect.Value, outInnerWasPointer bool, index []int, value string) error {
	oi := *outInner
	if outInnerWasPointer {
		oi = outInner.Elem()
	}
	return setField(oi.FieldByIndex(index), value)
}
