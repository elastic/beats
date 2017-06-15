// This file has been generated from 'fold_refl_sel.yml', do not edit
package gotype

import "reflect"

var _reflPrimitivesMapping = map[reflect.Type]reFoldFn{

	tBool: reFoldBool,
	reflect.SliceOf(tBool):        reFoldArrBool,
	reflect.MapOf(tString, tBool): reFoldMapBool,

	tString:                         reFoldString,
	reflect.SliceOf(tString):        reFoldArrString,
	reflect.MapOf(tString, tString): reFoldMapString,

	tUint: reFoldUint,
	reflect.SliceOf(tUint):        reFoldArrUint,
	reflect.MapOf(tString, tUint): reFoldMapUint,

	tUint8:                         reFoldUint8,
	reflect.SliceOf(tUint8):        reFoldArrUint8,
	reflect.MapOf(tString, tUint8): reFoldMapUint8,

	tUint16:                         reFoldUint16,
	reflect.SliceOf(tUint16):        reFoldArrUint16,
	reflect.MapOf(tString, tUint16): reFoldMapUint16,

	tUint32:                         reFoldUint32,
	reflect.SliceOf(tUint32):        reFoldArrUint32,
	reflect.MapOf(tString, tUint32): reFoldMapUint32,

	tUint64:                         reFoldUint64,
	reflect.SliceOf(tUint64):        reFoldArrUint64,
	reflect.MapOf(tString, tUint64): reFoldMapUint64,

	tInt: reFoldInt,
	reflect.SliceOf(tInt):        reFoldArrInt,
	reflect.MapOf(tString, tInt): reFoldMapInt,

	tInt8: reFoldInt8,
	reflect.SliceOf(tInt8):        reFoldArrInt8,
	reflect.MapOf(tString, tInt8): reFoldMapInt8,

	tInt16:                         reFoldInt16,
	reflect.SliceOf(tInt16):        reFoldArrInt16,
	reflect.MapOf(tString, tInt16): reFoldMapInt16,

	tInt32:                         reFoldInt32,
	reflect.SliceOf(tInt32):        reFoldArrInt32,
	reflect.MapOf(tString, tInt32): reFoldMapInt32,

	tInt64:                         reFoldInt64,
	reflect.SliceOf(tInt64):        reFoldArrInt64,
	reflect.MapOf(tString, tInt64): reFoldMapInt64,

	tFloat32:                         reFoldFloat32,
	reflect.SliceOf(tFloat32):        reFoldArrFloat32,
	reflect.MapOf(tString, tFloat32): reFoldMapFloat32,

	tFloat64:                         reFoldFloat64,
	reflect.SliceOf(tFloat64):        reFoldArrFloat64,
	reflect.MapOf(tString, tFloat64): reFoldMapFloat64,
}

func getReflectFoldPrimitive(t reflect.Type) reFoldFn {
	return _reflPrimitivesMapping[t]
}
