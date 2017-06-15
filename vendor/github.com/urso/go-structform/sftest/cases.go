package sftest

import structform "github.com/urso/go-structform"

var Samples = concatSamples(
	SamplesPrimitives,
	SamplesArr,
	SamplesObj,
	SamplesCombinations,
)

var SamplesPrimitives = []Recording{
	// simple primitives
	{NilRec{}},       // "null"
	{BoolRec{true}},  // "true"
	{BoolRec{false}}, // "false"
	{StringRec{"test"}},
	{StringRec{`test with " being special`}},
	{StringRec{""}},

	// int types
	{IntRec{8}},
	{IntRec{42}},
	{IntRec{100}},
	{IntRec{-90}},
	{IntRec{12345678}},
	{IntRec{-12345678}},
	{Int8Rec{8}},
	{Int8Rec{42}},
	{Int8Rec{100}},
	{Int8Rec{-90}},
	{Int16Rec{8}},
	{Int16Rec{42}},
	{Int16Rec{100}},
	{Int16Rec{-90}},
	{Int16Rec{500}},
	{Int16Rec{-500}},
	{Int32Rec{8}},
	{Int32Rec{42}},
	{Int32Rec{100}},
	{Int32Rec{-90}},
	{Int32Rec{500}},
	{Int32Rec{-500}},
	{Int32Rec{12345678}},
	{Int32Rec{-12345678}},
	{Int64Rec{8}},
	{Int64Rec{42}},
	{Int64Rec{100}},
	{Int64Rec{-90}},
	{Int64Rec{500}},
	{Int64Rec{-500}},
	{Int64Rec{123456781234}},
	{Int64Rec{-123456781234}},

	// uint types
	{UintRec{8}},
	{UintRec{42}},
	{UintRec{100}},
	{UintRec{12345678}},
	{Uint8Rec{8}},
	{Uint8Rec{42}},
	{Uint8Rec{100}},
	{ByteRec{8}},
	{ByteRec{42}},
	{ByteRec{100}},
	{Uint16Rec{8}},
	{Uint16Rec{42}},
	{Uint16Rec{100}},
	{Uint16Rec{500}},
	{Uint32Rec{8}},
	{Uint32Rec{42}},
	{Uint32Rec{100}},
	{Uint32Rec{500}},
	{Uint32Rec{12345678}},
	{Uint64Rec{8}},
	{Uint64Rec{42}},
	{Uint64Rec{100}},
	{Uint64Rec{500}},
	{Uint64Rec{123456781234}},

	// float types
	{Float32Rec{3.14}},
	{Float32Rec{-3.14}},
	{Float64Rec{3.14}},
	{Float64Rec{-3.14}},
}

var SamplesArr = []Recording{
	// empty arrays `[]`
	Arr(0, structform.AnyType),
	Arr(-1, structform.AnyType),

	// nested empty array `[[]]`
	Arr(1, structform.AnyType,
		Arr(0, structform.AnyType),
	),
	Arr(-1, structform.AnyType,
		Arr(0, structform.AnyType),
	),
	Arr(-1, structform.AnyType,
		Arr(-1, structform.AnyType),
	),

	// array with arbitrary values
	Arr(-1, structform.AnyType,
		NilRec{},
		BoolRec{true},
		BoolRec{false},
		IntRec{1},
		Int64Rec{12345678910},
		Float32Rec{3.14},
		Float64Rec{7e+09},
		StringRec{"test"},
	),
	Arr(2, structform.AnyType,
		Int8Rec{1},
		BoolRec{true},
	),
	{
		Int8ArrRec{[]int8{1, 2, 3}},
	},
	{
		StringArrRec{[]string{"a", "b", "c"}},
	},
}

var SamplesObj = []Recording{
	// empty object '{}'
	Obj(-1, structform.AnyType),
	Obj(0, structform.AnyType),

	Obj(-1, structform.AnyType,
		"a", NilRec{},
	),

	// objects
	Obj(-1, structform.AnyType,
		"a", StringRec{"test"}),
	Obj(1, structform.StringType,
		"a", StringRec{"test"}),
	Obj(-1, structform.AnyType,
		"a", BoolRec{true},
		"b", BoolRec{false},
	),
	Obj(2, structform.AnyType,
		"a", BoolRec{true},
		"b", BoolRec{false},
	),
	Obj(-1, structform.BoolType,
		"a", BoolRec{true},
		"b", BoolRec{false},
	),
	Obj(2, structform.BoolType,
		"a", BoolRec{true},
		"b", BoolRec{false},
	),
	Obj(-1, structform.AnyType,
		"a", UintRec{1},
		"b", Float64Rec{3.14},
		"c", StringRec{"test"},
		"d", BoolRec{true},
	),

	// typed objects
	{
		StringObjRec{map[string]string{
			"a": "test1",
			"b": "test2",
			"c": "test3",
		}},
	},
	{
		UintObjRec{map[string]uint{
			"a": 1,
			"b": 2,
			"c": 3,
		}},
	},
}

var SamplesCombinations = []Recording{
	// objects in array
	Arr(-1, structform.AnyType,
		Obj(-1, structform.AnyType)),
	Arr(1, structform.AnyType,
		Obj(0, structform.AnyType)),
	Arr(-1, structform.AnyType,
		Obj(-1, structform.AnyType,
			"a", IntRec{-1},
		),
		Obj(1, structform.UintType,
			"a", UintRec{1},
		),
	),
	Arr(2, structform.AnyType,
		Obj(-1, structform.AnyType,
			"a", IntRec{-1},
		),
		Obj(1, structform.UintType,
			"a", UintRec{1},
		),
	),

	// array in object
	Obj(-1, structform.AnyType,
		"a", Arr(3, structform.IntType,
			IntRec{1}, IntRec{2}, IntRec{3}),
	),
	Obj(1, structform.AnyType,
		"a", Arr(3, structform.IntType,
			IntRec{1}, IntRec{2}, IntRec{3}),
	),
	Obj(1, structform.AnyType,
		"a", Int8ArrRec{[]int8{1, 2, 3}},
	),
}
