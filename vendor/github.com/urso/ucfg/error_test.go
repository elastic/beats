package ucfg

import (
	"flag"
	"io/ioutil"
	"path"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

var updateFlag = flag.Bool("update", false, "Update the golden files.")

func TestErrorMessages(t *testing.T) {
	goldenPath := path.Join("testdata", "error", "message")

	c := New()
	cMeta := New()
	cNested := New()
	cNestedMeta := New()

	testMeta := &Meta{"test.source"}
	cMeta.metadata = testMeta
	cNestedMeta.metadata = testMeta

	testNestedCtx := context{field: "nested"}
	cNested.ctx = testNestedCtx
	cNestedMeta.ctx = testNestedCtx

	arr := &cfgArray{arr: make([]value, 3)}
	arrNested := &cfgArray{cfgPrimitive{ctx: testNestedCtx}, make([]value, 3)}
	arrMeta := &cfgArray{cfgPrimitive{metadata: testMeta}, make([]value, 3)}
	arrNestedMeta := &cfgArray{
		cfgPrimitive{metadata: testMeta, ctx: testNestedCtx},
		make([]value, 3)}

	tests := map[string]Error{
		"duplicate_wo_meta":        raiseDuplicateKey(c, "test"),
		"duplicate_w_meta":         raiseDuplicateKey(cMeta, "test"),
		"duplicate_nested_wo_meta": raiseDuplicateKey(cNested, "test"),
		"duplicate_nested_w_meta":  raiseDuplicateKey(cNestedMeta, "test"),

		"missing_wo_meta":        raiseMissing(c, "field"),
		"missing_w_meta":         raiseMissing(cMeta, "field"),
		"missing_nested_wo_meta": raiseMissing(cNested, "field"),
		"missing_nested_w_meta":  raiseMissing(cNestedMeta, "field"),

		"arr_missing_wo_meta":        raiseMissingArr(arr, 5),
		"arr_missing_w_meta":         raiseMissingArr(arrMeta, 5),
		"arr_missing_nested_wo_meta": raiseMissingArr(arrNested, 5),
		"arr_missing_nested_w_meta":  raiseMissingArr(arrNestedMeta, 5),

		"arr_oob_wo_meta":        raiseIndexOutOfBounds(arr, 5),
		"arr_oob_w_meta":         raiseIndexOutOfBounds(arrMeta, 5),
		"arr_oob_nested_wo_meta": raiseIndexOutOfBounds(arrNested, 5),
		"arr_oob_nested_w_meta":  raiseIndexOutOfBounds(arrNestedMeta, 5),

		"invalid_type_top_level": raiseInvalidTopLevelType(""),

		"invalid_type_unpack_wo_meta": raiseKeyInvalidTypeUnpack(
			reflect.TypeOf(map[int]interface{}{}), c),
		"invalid_type_unpack_w_meta": raiseKeyInvalidTypeUnpack(
			reflect.TypeOf(map[int]interface{}{}), cMeta),
		"invalid_type_unpack_nested_wo_meta": raiseKeyInvalidTypeUnpack(
			reflect.TypeOf(map[int]interface{}{}), cNested),
		"invalid_type_unpack_nested_w_meta": raiseKeyInvalidTypeUnpack(
			reflect.TypeOf(map[int]interface{}{}), cNestedMeta),

		"invalid_type_merge_wo_meta": raiseKeyInvalidTypeMerge(
			c, reflect.TypeOf(map[int]interface{}{})),
		"invalid_type_merge_w_meta": raiseKeyInvalidTypeMerge(
			cMeta, reflect.TypeOf(map[int]interface{}{})),
		"invalid_type_merge_nested_wo_meta": raiseKeyInvalidTypeMerge(
			cNested, reflect.TypeOf(map[int]interface{}{})),
		"invalid_type_merge_nested_w_meta": raiseKeyInvalidTypeMerge(
			cNestedMeta, reflect.TypeOf(map[int]interface{}{})),

		"squash_wo_meta": raiseSquashNeedsObject(
			c, options{}, "ABC", reflect.TypeOf("")),
		"squash_w_meta": raiseSquashNeedsObject(
			c, options{meta: testMeta}, "ABC", reflect.TypeOf("")),
		"squash_nested_wo_meta": raiseSquashNeedsObject(
			cNested, options{}, "ABC", reflect.TypeOf("")),
		"squash_nested_w_meta": raiseSquashNeedsObject(
			cNested, options{meta: testMeta}, "ABC", reflect.TypeOf("")),

		"inline_wo_meta": raiseInlineNeedsObject(
			c, "ABC", reflect.TypeOf("")),
		"inline_w_meta": raiseInlineNeedsObject(
			cMeta, "ABC", reflect.TypeOf("")),
		"inline_nested_wo_meta": raiseInlineNeedsObject(
			cNested, "ABC", reflect.TypeOf("")),
		"inline_nested_w_meta": raiseInlineNeedsObject(
			cNestedMeta, "ABC", reflect.TypeOf("")),

		"unsupported_input_type_wo_meta": raiseUnsupportedInputType(
			context{}, options{}, reflect.ValueOf(1)),
		"unsupported_input_type_w_meta": raiseUnsupportedInputType(
			context{}, options{meta: testMeta}, reflect.ValueOf(1)),
		"unsupported_input_type_nested_wo_meta": raiseUnsupportedInputType(
			testNestedCtx, options{}, reflect.ValueOf(1)),
		"unsupported_input_type_nested_w_meta": raiseUnsupportedInputType(
			testNestedCtx, options{meta: testMeta}, reflect.ValueOf(1)),

		"nil_value_error":  raiseNil(ErrNilValue),
		"nil_config_error": raiseNil(ErrNilConfig),

		"pointer_required": raisePointerRequired(reflect.ValueOf(1)),

		"to_type_not_supported_wo_meta": raiseToTypeNotSupported(
			newInt(context{}, nil, 1), reflect.TypeOf(struct{}{})),
		"to_type_not_supported_w_meta": raiseToTypeNotSupported(
			newInt(context{}, testMeta, 1), reflect.TypeOf(struct{}{})),
		"to_type_not_supported_nested_wo_meta": raiseToTypeNotSupported(
			newInt(testNestedCtx, nil, 1), reflect.TypeOf(struct{}{})),
		"to_type_not_supported_nested_w_meta": raiseToTypeNotSupported(
			newInt(testNestedCtx, testMeta, 1), reflect.TypeOf(struct{}{})),

		"array_size_wo_meta": raiseArraySize(reflect.TypeOf([10]int{}), arr),
		"array_size_w_meta": raiseArraySize(
			reflect.TypeOf([10]int{}), arrMeta),
		"array_size_nested_wo_meta": raiseArraySize(
			reflect.TypeOf([10]int{}), arrNested),
		"array_size_nested_w_meta": raiseArraySize(
			reflect.TypeOf([10]int{}), arrNestedMeta),

		"conversion_wo_meta": raiseConversion(
			newInt(context{}, nil, 1), ErrTypeMismatch, "bool"),
		"conversion_w_meta": raiseConversion(
			newInt(context{}, testMeta, 1), ErrTypeMismatch, "bool"),
		"conversion_nested_wo_meta": raiseConversion(
			newInt(testNestedCtx, nil, 1), ErrTypeMismatch, "bool"),
		"conversion_nested_w_meta": raiseConversion(
			newInt(testNestedCtx, testMeta, 1), ErrTypeMismatch, "bool"),

		"expected_object_wo_meta": raiseExpectedObject(
			newInt(context{}, nil, 1)),
		"expected_object_w_meta": raiseExpectedObject(
			newInt(context{}, testMeta, 1)),
		"expected_object_nested_wo_meta": raiseExpectedObject(
			newInt(testNestedCtx, nil, 1)),
		"expected_object_nested_w_meta": raiseExpectedObject(
			newInt(testNestedCtx, testMeta, 1)),
	}

	for name, result := range tests {
		t.Logf("Test error message for: %v", name)

		message := result.Message()
		goldenFile := path.Join(goldenPath, name+".golden")

		if updateFlag != nil && *updateFlag {
			t.Logf("writing golden file: %v", goldenFile)
			t.Logf("%v", message)
			t.Log("")
			err := ioutil.WriteFile(goldenFile, []byte(message), 0666)
			if err != nil {
				t.Fatalf("Failed to write golden file ('%v'): %v", goldenFile, err)
			}
		}

		tmp, err := ioutil.ReadFile(goldenFile)
		if err != nil {
			t.Fatalf("Failed to read golden file ('%v'): %v", goldenFile, err)
		}

		golden := string(tmp)
		assert.Equal(t, golden, message)
	}
}
