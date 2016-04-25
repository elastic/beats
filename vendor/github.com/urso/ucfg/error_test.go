package ucfg

import (
	"errors"
	"flag"
	"io/ioutil"
	"path"
	"reflect"
	"testing"
	"time"

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

	_, timeErr := time.ParseDuration("1 hour")

	tests := map[string]Error{
		"duplicate_wo_meta":        raiseDuplicateKey(c, "test"),
		"duplicate_w_meta":         raiseDuplicateKey(cMeta, "test"),
		"duplicate_nested_wo_meta": raiseDuplicateKey(cNested, "test"),
		"duplicate_nested_w_meta":  raiseDuplicateKey(cNestedMeta, "test"),

		"missing_wo_meta":        raiseMissing(c, "field"),
		"missing_w_meta":         raiseMissing(cMeta, "field"),
		"missing_nested_wo_meta": raiseMissing(cNested, "field"),
		"missing_nested_w_meta":  raiseMissing(cNestedMeta, "field"),

		"arr_missing_wo_meta":        raiseMissingArr(context{}, nil, 5),
		"arr_missing_w_meta":         raiseMissingArr(context{}, testMeta, 5),
		"arr_missing_nested_wo_meta": raiseMissingArr(testNestedCtx, nil, 5),
		"arr_missing_nested_w_meta":  raiseMissingArr(testNestedCtx, testMeta, 5),

		"arr_oob_wo_meta":        raiseIndexOutOfBounds(cfgSub{c}, 5),
		"arr_oob_w_meta":         raiseIndexOutOfBounds(cfgSub{cMeta}, 5),
		"arr_oob_nested_wo_meta": raiseIndexOutOfBounds(cfgSub{cNested}, 5),
		"arr_oob_nested_w_meta":  raiseIndexOutOfBounds(cfgSub{cNestedMeta}, 5),

		"invalid_duration_wo_meta": raiseInvalidDuration(newString(
			context{field: "timeout"}, nil, ""), timeErr),
		"invalid_duration_w_meta": raiseInvalidDuration(newString(
			context{field: "timeout"}, testMeta, ""), timeErr),

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
			context{}, nil, reflect.ValueOf(1)),
		"unsupported_input_type_w_meta": raiseUnsupportedInputType(
			context{}, testMeta, reflect.ValueOf(1)),
		"unsupported_input_type_nested_wo_meta": raiseUnsupportedInputType(
			testNestedCtx, nil, reflect.ValueOf(1)),
		"unsupported_input_type_nested_w_meta": raiseUnsupportedInputType(
			testNestedCtx, testMeta, reflect.ValueOf(1)),

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

		"array_size_wo_meta": raiseArraySize(
			context{}, nil, 3, 10),
		"array_size_w_meta": raiseArraySize(
			context{}, testMeta, 3, 10),
		"array_size_nested_wo_meta": raiseArraySize(
			testNestedCtx, nil, 3, 10),
		"array_size_nested_w_meta": raiseArraySize(
			testNestedCtx, testMeta, 3, 10),

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

		"validation_wo_meta": raiseValidation(
			context{}, nil, errors.New("invalid value")),
		"validation_w_meta": raiseValidation(
			context{}, testMeta, errors.New("invalid value")),
		"validation_nested_wo_meta": raiseValidation(
			testNestedCtx, nil, errors.New("invalid value")),
		"validation_nested_w_meta": raiseValidation(
			testNestedCtx, testMeta, errors.New("invalid value")),
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
