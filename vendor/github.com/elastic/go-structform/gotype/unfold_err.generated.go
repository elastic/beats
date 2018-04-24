// This file has been generated from 'unfold_err.yml', do not edit
package gotype

import structform "github.com/elastic/go-structform"

type unfolderErrArrayStart struct{}

type unfolderErrExpectKey struct{}

type unfolderErrObjectStart struct{}

type unfolderErrUnknown struct{}

type unfolderNoTarget struct{}

func (*unfolderErrArrayStart) OnNil(*unfoldCtx) error               { return errExpectedArray }
func (*unfolderErrArrayStart) OnBool(*unfoldCtx, bool) error        { return errExpectedArray }
func (*unfolderErrArrayStart) OnString(*unfoldCtx, string) error    { return errExpectedArray }
func (*unfolderErrArrayStart) OnStringRef(*unfoldCtx, []byte) error { return errExpectedArray }
func (*unfolderErrArrayStart) OnInt8(*unfoldCtx, int8) error        { return errExpectedArray }
func (*unfolderErrArrayStart) OnInt16(*unfoldCtx, int16) error      { return errExpectedArray }
func (*unfolderErrArrayStart) OnInt32(*unfoldCtx, int32) error      { return errExpectedArray }
func (*unfolderErrArrayStart) OnInt64(*unfoldCtx, int64) error      { return errExpectedArray }
func (*unfolderErrArrayStart) OnInt(*unfoldCtx, int) error          { return errExpectedArray }
func (*unfolderErrArrayStart) OnByte(*unfoldCtx, byte) error        { return errExpectedArray }
func (*unfolderErrArrayStart) OnUint8(*unfoldCtx, uint8) error      { return errExpectedArray }
func (*unfolderErrArrayStart) OnUint16(*unfoldCtx, uint16) error    { return errExpectedArray }
func (*unfolderErrArrayStart) OnUint32(*unfoldCtx, uint32) error    { return errExpectedArray }
func (*unfolderErrArrayStart) OnUint64(*unfoldCtx, uint64) error    { return errExpectedArray }
func (*unfolderErrArrayStart) OnUint(*unfoldCtx, uint) error        { return errExpectedArray }
func (*unfolderErrArrayStart) OnFloat32(*unfoldCtx, float32) error  { return errExpectedArray }
func (*unfolderErrArrayStart) OnFloat64(*unfoldCtx, float64) error  { return errExpectedArray }
func (*unfolderErrArrayStart) OnArrayStart(*unfoldCtx, int, structform.BaseType) error {
	return errExpectedArray
}
func (*unfolderErrArrayStart) OnArrayFinished(*unfoldCtx) error  { return errExpectedArray }
func (*unfolderErrArrayStart) OnChildArrayDone(*unfoldCtx) error { return errExpectedArray }
func (*unfolderErrArrayStart) OnObjectStart(*unfoldCtx, int, structform.BaseType) error {
	return errExpectedArray
}
func (*unfolderErrArrayStart) OnObjectFinished(*unfoldCtx) error  { return errExpectedArray }
func (*unfolderErrArrayStart) OnKey(*unfoldCtx, string) error     { return errExpectedArray }
func (*unfolderErrArrayStart) OnKeyRef(*unfoldCtx, []byte) error  { return errExpectedArray }
func (*unfolderErrArrayStart) OnChildObjectDone(*unfoldCtx) error { return errExpectedArray }

func (*unfolderErrExpectKey) OnNil(*unfoldCtx) error               { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnBool(*unfoldCtx, bool) error        { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnString(*unfoldCtx, string) error    { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnStringRef(*unfoldCtx, []byte) error { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnInt8(*unfoldCtx, int8) error        { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnInt16(*unfoldCtx, int16) error      { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnInt32(*unfoldCtx, int32) error      { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnInt64(*unfoldCtx, int64) error      { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnInt(*unfoldCtx, int) error          { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnByte(*unfoldCtx, byte) error        { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnUint8(*unfoldCtx, uint8) error      { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnUint16(*unfoldCtx, uint16) error    { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnUint32(*unfoldCtx, uint32) error    { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnUint64(*unfoldCtx, uint64) error    { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnUint(*unfoldCtx, uint) error        { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnFloat32(*unfoldCtx, float32) error  { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnFloat64(*unfoldCtx, float64) error  { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnArrayStart(*unfoldCtx, int, structform.BaseType) error {
	return errExpectedObjectKey
}
func (*unfolderErrExpectKey) OnArrayFinished(*unfoldCtx) error  { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnChildArrayDone(*unfoldCtx) error { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnObjectStart(*unfoldCtx, int, structform.BaseType) error {
	return errExpectedObjectKey
}
func (*unfolderErrExpectKey) OnObjectFinished(*unfoldCtx) error  { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnKey(*unfoldCtx, string) error     { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnKeyRef(*unfoldCtx, []byte) error  { return errExpectedObjectKey }
func (*unfolderErrExpectKey) OnChildObjectDone(*unfoldCtx) error { return errExpectedObjectKey }

func (*unfolderErrObjectStart) OnNil(*unfoldCtx) error               { return errExpectedObject }
func (*unfolderErrObjectStart) OnBool(*unfoldCtx, bool) error        { return errExpectedObject }
func (*unfolderErrObjectStart) OnString(*unfoldCtx, string) error    { return errExpectedObject }
func (*unfolderErrObjectStart) OnStringRef(*unfoldCtx, []byte) error { return errExpectedObject }
func (*unfolderErrObjectStart) OnInt8(*unfoldCtx, int8) error        { return errExpectedObject }
func (*unfolderErrObjectStart) OnInt16(*unfoldCtx, int16) error      { return errExpectedObject }
func (*unfolderErrObjectStart) OnInt32(*unfoldCtx, int32) error      { return errExpectedObject }
func (*unfolderErrObjectStart) OnInt64(*unfoldCtx, int64) error      { return errExpectedObject }
func (*unfolderErrObjectStart) OnInt(*unfoldCtx, int) error          { return errExpectedObject }
func (*unfolderErrObjectStart) OnByte(*unfoldCtx, byte) error        { return errExpectedObject }
func (*unfolderErrObjectStart) OnUint8(*unfoldCtx, uint8) error      { return errExpectedObject }
func (*unfolderErrObjectStart) OnUint16(*unfoldCtx, uint16) error    { return errExpectedObject }
func (*unfolderErrObjectStart) OnUint32(*unfoldCtx, uint32) error    { return errExpectedObject }
func (*unfolderErrObjectStart) OnUint64(*unfoldCtx, uint64) error    { return errExpectedObject }
func (*unfolderErrObjectStart) OnUint(*unfoldCtx, uint) error        { return errExpectedObject }
func (*unfolderErrObjectStart) OnFloat32(*unfoldCtx, float32) error  { return errExpectedObject }
func (*unfolderErrObjectStart) OnFloat64(*unfoldCtx, float64) error  { return errExpectedObject }
func (*unfolderErrObjectStart) OnArrayStart(*unfoldCtx, int, structform.BaseType) error {
	return errExpectedObject
}
func (*unfolderErrObjectStart) OnArrayFinished(*unfoldCtx) error  { return errExpectedObject }
func (*unfolderErrObjectStart) OnChildArrayDone(*unfoldCtx) error { return errExpectedObject }
func (*unfolderErrObjectStart) OnObjectStart(*unfoldCtx, int, structform.BaseType) error {
	return errExpectedObject
}
func (*unfolderErrObjectStart) OnObjectFinished(*unfoldCtx) error  { return errExpectedObject }
func (*unfolderErrObjectStart) OnKey(*unfoldCtx, string) error     { return errExpectedObject }
func (*unfolderErrObjectStart) OnKeyRef(*unfoldCtx, []byte) error  { return errExpectedObject }
func (*unfolderErrObjectStart) OnChildObjectDone(*unfoldCtx) error { return errExpectedObject }

func (*unfolderErrUnknown) OnNil(*unfoldCtx) error               { return errUnsupported }
func (*unfolderErrUnknown) OnBool(*unfoldCtx, bool) error        { return errUnsupported }
func (*unfolderErrUnknown) OnString(*unfoldCtx, string) error    { return errUnsupported }
func (*unfolderErrUnknown) OnStringRef(*unfoldCtx, []byte) error { return errUnsupported }
func (*unfolderErrUnknown) OnInt8(*unfoldCtx, int8) error        { return errUnsupported }
func (*unfolderErrUnknown) OnInt16(*unfoldCtx, int16) error      { return errUnsupported }
func (*unfolderErrUnknown) OnInt32(*unfoldCtx, int32) error      { return errUnsupported }
func (*unfolderErrUnknown) OnInt64(*unfoldCtx, int64) error      { return errUnsupported }
func (*unfolderErrUnknown) OnInt(*unfoldCtx, int) error          { return errUnsupported }
func (*unfolderErrUnknown) OnByte(*unfoldCtx, byte) error        { return errUnsupported }
func (*unfolderErrUnknown) OnUint8(*unfoldCtx, uint8) error      { return errUnsupported }
func (*unfolderErrUnknown) OnUint16(*unfoldCtx, uint16) error    { return errUnsupported }
func (*unfolderErrUnknown) OnUint32(*unfoldCtx, uint32) error    { return errUnsupported }
func (*unfolderErrUnknown) OnUint64(*unfoldCtx, uint64) error    { return errUnsupported }
func (*unfolderErrUnknown) OnUint(*unfoldCtx, uint) error        { return errUnsupported }
func (*unfolderErrUnknown) OnFloat32(*unfoldCtx, float32) error  { return errUnsupported }
func (*unfolderErrUnknown) OnFloat64(*unfoldCtx, float64) error  { return errUnsupported }
func (*unfolderErrUnknown) OnArrayStart(*unfoldCtx, int, structform.BaseType) error {
	return errUnsupported
}
func (*unfolderErrUnknown) OnArrayFinished(*unfoldCtx) error  { return errUnsupported }
func (*unfolderErrUnknown) OnChildArrayDone(*unfoldCtx) error { return errUnsupported }
func (*unfolderErrUnknown) OnObjectStart(*unfoldCtx, int, structform.BaseType) error {
	return errUnsupported
}
func (*unfolderErrUnknown) OnObjectFinished(*unfoldCtx) error  { return errUnsupported }
func (*unfolderErrUnknown) OnKey(*unfoldCtx, string) error     { return errUnsupported }
func (*unfolderErrUnknown) OnKeyRef(*unfoldCtx, []byte) error  { return errUnsupported }
func (*unfolderErrUnknown) OnChildObjectDone(*unfoldCtx) error { return errUnsupported }

func (*unfolderNoTarget) OnNil(*unfoldCtx) error               { return errNotInitialized }
func (*unfolderNoTarget) OnBool(*unfoldCtx, bool) error        { return errNotInitialized }
func (*unfolderNoTarget) OnString(*unfoldCtx, string) error    { return errNotInitialized }
func (*unfolderNoTarget) OnStringRef(*unfoldCtx, []byte) error { return errNotInitialized }
func (*unfolderNoTarget) OnInt8(*unfoldCtx, int8) error        { return errNotInitialized }
func (*unfolderNoTarget) OnInt16(*unfoldCtx, int16) error      { return errNotInitialized }
func (*unfolderNoTarget) OnInt32(*unfoldCtx, int32) error      { return errNotInitialized }
func (*unfolderNoTarget) OnInt64(*unfoldCtx, int64) error      { return errNotInitialized }
func (*unfolderNoTarget) OnInt(*unfoldCtx, int) error          { return errNotInitialized }
func (*unfolderNoTarget) OnByte(*unfoldCtx, byte) error        { return errNotInitialized }
func (*unfolderNoTarget) OnUint8(*unfoldCtx, uint8) error      { return errNotInitialized }
func (*unfolderNoTarget) OnUint16(*unfoldCtx, uint16) error    { return errNotInitialized }
func (*unfolderNoTarget) OnUint32(*unfoldCtx, uint32) error    { return errNotInitialized }
func (*unfolderNoTarget) OnUint64(*unfoldCtx, uint64) error    { return errNotInitialized }
func (*unfolderNoTarget) OnUint(*unfoldCtx, uint) error        { return errNotInitialized }
func (*unfolderNoTarget) OnFloat32(*unfoldCtx, float32) error  { return errNotInitialized }
func (*unfolderNoTarget) OnFloat64(*unfoldCtx, float64) error  { return errNotInitialized }
func (*unfolderNoTarget) OnArrayStart(*unfoldCtx, int, structform.BaseType) error {
	return errNotInitialized
}
func (*unfolderNoTarget) OnArrayFinished(*unfoldCtx) error  { return errNotInitialized }
func (*unfolderNoTarget) OnChildArrayDone(*unfoldCtx) error { return errNotInitialized }
func (*unfolderNoTarget) OnObjectStart(*unfoldCtx, int, structform.BaseType) error {
	return errNotInitialized
}
func (*unfolderNoTarget) OnObjectFinished(*unfoldCtx) error  { return errNotInitialized }
func (*unfolderNoTarget) OnKey(*unfoldCtx, string) error     { return errNotInitialized }
func (*unfolderNoTarget) OnKeyRef(*unfoldCtx, []byte) error  { return errNotInitialized }
func (*unfolderNoTarget) OnChildObjectDone(*unfoldCtx) error { return errNotInitialized }
