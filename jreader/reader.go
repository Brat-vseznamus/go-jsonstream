package jreader

import (
	"fmt"
	"strconv"
)

// Reader is a high-level API for reading JSON data sequentially.
//
// It is designed to make writing custom unmarshallers for application types as convenient as
// possible. The general usage pattern is as follows:
//
// - Values are parsed in the order that they appear.
//
// - In general, the caller should know what data type is expected. Since it is common for
// properties to be nullable, the methods for reading scalar types have variants for allowing
// a null instead of the specified type. If the type is completely unknown, use Any.
//
// - For reading array or object structures, the Array and Object methods return a struct that
// keeps track of additional reader state while that structure is being parsed.
//
// - If any method encounters an error (due to either malformed JSON, or well-formed JSON that
// did not match the caller's data type expectations), the Reader permanently enters a failed
// state and remembers that error; all subsequent method calls will return the same error and no
// more parsing will happen. This means that the caller does not necessarily have to check the
// error return Value of any individual method, although it can.
type Reader struct {
	tr                tokenReader
	awaitingReadValue bool // used by ArrayState & ObjectState
	err               error
}

// Reset drops all states and reset all buffers to nils
func (r *Reader) Reset(data []byte) {
	r.err = nil
	r.awaitingReadValue = false
	r.tr.Reset(data)
}

// Error returns the first error that the Reader encountered, if the Reader is in a failed state,
// or nil if it is still in a good state.
func (r *Reader) Error() error {
	return r.err
}

// RequireEOF returns nil if all the input has been consumed (not counting whitespace), or an
// error if not.
func (r *Reader) RequireEOF() error {
	if !r.tr.EOF() {
		return SyntaxError{Message: errMsgDataAfterEnd, Offset: r.tr.LastPos()}
	}
	return nil
}

// AddError sets the Reader's error value and puts it into a failed state. If the parameter is nil
// or the Reader was already in a failed state, it does nothing.
func (r *Reader) AddError(err error) {
	if r.err == nil {
		r.err = err
	}
}

// ReplaceError sets the Reader's error value and puts it into a failed state, replacing any
// previously reported error. If the parameter is nil, it does nothing (a failed state cannot be
// changed to a non-failed state).
func (r *Reader) ReplaceError(err error) {
	if err != nil {
		r.err = err
	}
}

// Null attempts to read a null value, returning an error if the next token is not a null.
func (r *Reader) Null() error {
	r.awaitingReadValue = false
	if r.err != nil {
		return r.err
	}
	isNull, err := r.tr.Null()
	if isNull || err != nil {
		return err
	}
	return r.typeErrorForCurrentToken(NullValue, false)
}

// Bool attempts to read a boolean value.
//
// If there is a parsing error, or the next value is not a boolean, the return value is false
// and the Reader enters a failed state, which you can detect with Error().
func (r *Reader) Bool() bool {
	r.awaitingReadValue = false
	if r.err != nil {
		return false
	}
	val, err := r.tr.Bool()
	if err != nil {
		r.err = err
		return false
	}
	return val
}

// BoolOrNull attempts to read either a boolean value or a null. In the case of a boolean, the return
// values are (value, true); for a null, they are (false, false).
//
// If there is a parsing error, or the next value is neither a boolean nor a null, the return values
// are (false, false) and the Reader enters a failed state, which you can detect with Error().
func (r *Reader) BoolOrNull() (value bool, nonNull bool) {
	r.awaitingReadValue = false
	if r.err != nil {
		return false, false
	}
	isNull, err := r.tr.Null()
	if isNull || err != nil {
		r.err = err
		return false, false
	}
	val, err := r.tr.Bool()
	if err != nil {
		r.err = typeErrorForNullableValue(err)
		return false, false
	}
	return val, true
}

func (r *Reader) NumberProps() *NumberProps {
	r.awaitingReadValue = false
	if r.err != nil {
		return nil
	}
	val, err := r.tr.Number()
	if err != nil {
		r.err = err
		return nil
	}
	return val
}

func (r *Reader) NumberPropsOrNull() (*NumberProps, bool) {
	r.awaitingReadValue = false
	if r.err != nil {
		return nil, false
	}
	isNull, err := r.tr.Null()
	if isNull || err != nil {
		r.err = err
		return nil, false
	}
	val, err := r.tr.Number()
	if err != nil {
		r.err = typeErrorForNullableValue(err)
		return nil, false
	}
	return val, true
}

func (r *Reader) Number() []byte {
	r.awaitingReadValue = false
	if r.err != nil {
		return nil
	}
	val, err := r.tr.Number()
	if err != nil {
		r.err = err
		return nil
	}
	return val.raw
}

func (r *Reader) NumberOrNull() ([]byte, bool) {
	r.awaitingReadValue = false
	if r.err != nil {
		return nil, false
	}
	isNull, err := r.tr.Null()
	if isNull || err != nil {
		r.err = err
		return nil, false
	}
	val, err := r.tr.Number()
	if err != nil {
		r.err = typeErrorForNullableValue(err)
		return nil, false
	}
	return val.raw, true
}

func (r *Reader) UInt64() uint64 {
	r.awaitingReadValue = false
	if r.err != nil {
		return 0
	}
	val, err := r.tr.Number()
	if err != nil {
		r.err = err
		return 0
	}
	if r.IsNumbersRaw() {
		result, _ := strconv.ParseUint(string(val.raw), 10, 64)
		return result
	} else {
		result, err := val.UInt64()
		if err != nil {
			r.err = err
			return 0
		}
		return result
	}
}

func (r *Reader) UInt64OrNull() (uint64, bool) {
	r.awaitingReadValue = false
	if r.err != nil {
		return 0, false
	}
	isNull, err := r.tr.Null()
	if isNull || err != nil {
		r.err = err
		return 0, false
	}
	val, err := r.tr.Number()
	if err != nil {
		r.err = typeErrorForNullableValue(err)
		return 0, false
	}
	if r.IsNumbersRaw() {
		result, err := strconv.ParseUint(string(val.raw), 10, 64)
		return result, err == nil
	} else {
		result, err := val.UInt64()
		if err != nil {
			r.err = err
			return 0, false
		}
		return result, true
	}
}

// Int64 attempts to read a numeric value and returns it as an int.
//
// If there is a parsing error, or the next value is not a number, the return value is zero and
// the Reader enters a failed state, which you can detect with Error(). Non-numeric types are never
// converted to numbers.
func (r *Reader) Int64() int64 {
	r.awaitingReadValue = false
	if r.err != nil {
		return 0
	}
	val, err := r.tr.Number()
	if err != nil {
		r.err = err
		return 0
	}
	if r.IsNumbersRaw() {
		result, _ := strconv.ParseInt(string(val.raw), 10, 64)
		return result
	} else {
		result, err := val.Int64()
		if err != nil {
			r.err = err
			return 0
		}
		return result
	}
}

// Int64OrNull attempts to read either an integer numeric value or a null. In the case of a number, the
// return values are (value, true); for a null, they are (0, false).
//
// If there is a parsing error, or the next value is neither a number nor a null, the return values
// are (0, false) and the Reader enters a failed state, which you can detect with Error().
func (r *Reader) Int64OrNull() (int64, bool) {
	r.awaitingReadValue = false
	if r.err != nil {
		return 0, false
	}
	isNull, err := r.tr.Null()
	if isNull || err != nil {
		r.err = err
		return 0, false
	}
	val, err := r.tr.Number()
	if err != nil {
		r.err = typeErrorForNullableValue(err)
		return 0, false
	}
	if r.IsNumbersRaw() {
		result, err := strconv.ParseInt(string(val.raw), 10, 64)
		return result, err == nil
	} else {
		result, err := val.Int64()
		if err != nil {
			r.err = err
			return 0, false
		}
		return result, true
	}
}

// Float64 attempts to read a numeric value and returns it as a float64.
//
// If there is a parsing error, or the next value is not a number, the return value is zero and
// the Reader enters a failed state, which you can detect with Error(). Non-numeric types are never
// converted to numbers.
func (r *Reader) Float64() float64 {
	r.awaitingReadValue = false
	if r.err != nil {
		return 0
	}
	val, err := r.tr.Number()
	if err != nil {
		r.err = err
		return 0
	}
	if r.IsNumbersRaw() {
		result, _ := strconv.ParseFloat(string(val.raw), 64)
		return result
	} else {
		result, err := val.Float64()
		if err != nil {
			r.err = err
			return 0
		}
		return result
	}
}

// Float64OrNull attempts to read either a numeric value or a null. In the case of a number, the
// return values are (value, true); for a null, they are (0, false).
//
// If there is a parsing error, or the next value is neither a number nor a null, the return values
// are (0, false) and the Reader enters a failed state, which you can detect with Error().
func (r *Reader) Float64OrNull() (float64, bool) {
	r.awaitingReadValue = false
	if r.err != nil {
		return 0, false
	}
	isNull, err := r.tr.Null()
	if isNull || err != nil {
		r.err = err
		return 0, false
	}
	val, err := r.tr.Number()
	if err != nil {
		r.err = typeErrorForNullableValue(err)
		return 0, false
	}
	if r.IsNumbersRaw() {
		result, err := strconv.ParseFloat(string(val.raw), 64)
		return result, err == nil
	} else {
		result, err := val.Float64()
		if err != nil {
			r.err = err
			return 0, false
		}
		return result, true
	}
}

// String attempts to read a string value.
//
// If there is a parsing error, or the next value is not a string, the return value is "" and
// the Reader enters a failed state, which you can detect with Error(). Types other than string
// are never converted to strings.
func (r *Reader) String() []byte {
	r.awaitingReadValue = false
	if r.err != nil {
		return []byte("")
	}
	val, err := r.tr.String()
	if err != nil {
		r.err = err
		return []byte("")
	}
	return val
}

// StringOrNull attempts to read either a string value or a null. In the case of a string, the
// return values are (value, true); for a null, they are ("", false).
//
// If there is a parsing error, or the next value is neither a string nor a null, the return values
// are ("", false) and the Reader enters a failed state, which you can detect with Error().
func (r *Reader) StringOrNull() ([]byte, bool) {
	r.awaitingReadValue = false
	if r.err != nil {
		return []byte(""), false
	}
	isNull, err := r.tr.Null()
	if isNull || err != nil {
		r.err = err
		return []byte(""), false
	}
	val, err := r.tr.String()
	if err != nil {
		r.err = typeErrorForNullableValue(err)
		return []byte(""), false
	}
	return val, true
}

// Array attempts to begin reading a JSON array value. If successful, the return value will be an
// ArrayState containing the necessary state for iterating through the array elements.
//
// The ArrayState is used only for the iteration state; to read the value of each array element, you
// will still use the Reader's methods.
//
// If there is a parsing error, or the next value is not an array, the returned ArrayState is a stub
// whose Next() method always returns false, and the Reader enters a failed state, which you can
// detect with Error().
//
// See ArrayState for example code.
func (r *Reader) Array() ArrayState {
	return r.tryArray(false)
}

// ArrayOrNull attempts to either begin reading an JSON array value, or read a null. In the case of an
// array, the return value will be an ArrayState containing the necessary state for iterating through
// the array elements; the ArrayState's IsDefined() method will return true. In the case of a null, the
// returned ArrayState will be a stub whose Next() and IsDefined() methods always returns false.
//
// The ArrayState is used only for the iteration state; to read the value of each array element, you
// will still use the Reader's methods.
//
// If there is a parsing error, or the next value is neither an array nor a null, the return value is
// the same as for a null but the Reader enters a failed state, which you can detect with Error().
//
// See ArrayState for example code.
func (r *Reader) ArrayOrNull() ArrayState {
	return r.tryArray(true)
}

func (r *Reader) tryArray(allowNull bool) ArrayState {
	r.awaitingReadValue = false
	if r.err != nil {
		return ArrayState{}
	}
	if allowNull {
		isNull, err := r.tr.Null()
		if err != nil {
			r.err = err
			return ArrayState{}
		}
		if isNull {
			return ArrayState{}
		}
	}
	gotDelim, err := r.tr.Delimiter('[')
	if err != nil {
		r.err = err
		return ArrayState{}
	}
	if gotDelim {
		if r.tr.options.lazyRead {
			return ArrayState{r: r, arrayIndex: r.tr.structBuffer.Pos}
		} else {
			return ArrayState{r: r}
		}
	}
	r.err = r.typeErrorForCurrentToken(ArrayValue, allowNull)
	return ArrayState{}
}

// Object attempts to begin reading a JSON object value. If successful, the return value will be an
// ObjectState containing the necessary state for iterating through the object properties.
//
// The ObjectState is used only for the iteration state; to read the value of each property, you
// will still use the Reader's methods.
//
// If there is a parsing error, or the next value is not an object, the returned ObjectState is a stub
// whose Next() method always returns false, and the Reader enters a failed state, which you can
// detect with Error().
//
// See ObjectState for example code.
func (r *Reader) Object() ObjectState {
	return r.tryObject(false)
}

// ObjectOrNull attempts to either begin reading an JSON object value, or read a null. In the case of an
// object, the return value will be an ObjectState containing the necessary state for iterating through
// the object properties; the ObjectState's IsDefined() method will return true. In the case of a null,
// the returned ObjectState will be a stub whose Next() and IsDefined() methods always returns false.
//
// The ObjectState is used only for the iteration state; to read the value of each property, you
// will still use the Reader's methods.
//
// If there is a parsing error, or the next value is neither an object nor a null, the return value is
// the same as for a null but the Reader enters a failed state, which you can detect with Error().
//
// See ObjectState for example code.
func (r *Reader) ObjectOrNull() ObjectState {
	return r.tryObject(true)
}

func (r *Reader) tryObject(allowNull bool) ObjectState {
	r.awaitingReadValue = false
	if r.err != nil {
		return ObjectState{}
	}
	if allowNull {
		isNull, err := r.tr.Null()
		if err != nil || isNull {
			r.err = err
			return ObjectState{}
		}
	}
	gotDelim, err := r.tr.Delimiter('{')
	if err != nil {
		r.err = err
		return ObjectState{}
	}
	if gotDelim {
		if r.tr.options.lazyRead {
			return ObjectState{r: r, objectIndex: r.tr.structBuffer.Pos}
		} else {
			return ObjectState{r: r}
		}
	}
	r.err = r.typeErrorForCurrentToken(ObjectValue, allowNull)
	return ObjectState{}
}

// Any reads a single value of any type, if it is a scalar value or a null, or prepares to read
// the value if it is an array or object.
//
// The returned AnyValue's Kind field indicates the value type. If it is BoolValue, NumberValue,
// or StringValue, check the corresponding Bool, Number, or String property. If it is ArrayValue
// or ObjectValue, the AnyValue's Array or Object field has been initialized with an ArrayState or
// ObjectState just as if you had called the Reader's Array or Object method.
//
// If there is a parsing error, the return value is the same as for a null and the Reader enters
// a failed state, which you can detect with Error().
func (r *Reader) Any() *AnyValue {
	r.awaitingReadValue = false
	if r.err != nil {
		return nil
	}
	v, err := r.tr.Any()
	if err != nil {
		r.err = err
		return nil
	}
	switch v.Kind {
	case BoolValue:
		return v
	case NumberValue:
		return v
	case StringValue:
		return v
	case ArrayValue:
		v.Array.arrayIndex = r.tr.structBuffer.Pos
		v.Array.r = r
		return v
	case ObjectValue:
		v.Object.objectIndex = r.tr.structBuffer.Pos
		v.Object.r = r
		return v
	default:
		return v
	}
}

// SkipValue consumes and discards the next JSON value of any type. For an array or object value, it
// recurses to also consume and discard all array elements or object properties.
func (r *Reader) SkipValue() error {
	if r.tr.options.lazyRead {
		skipped := r.tr.structBuffer.SkipSubTree()
		if skipped {
			return nil
		} else {
			return fmt.Errorf("subtree can't be skipped")
		}
	} else {
		r.awaitingReadValue = false
		if r.err != nil {
			return r.err
		}
		v := r.Any()
		if v.Kind == ArrayValue {
			arr := v.Array
			for arr.Next() {
			}
		} else if v.Kind == ObjectValue {
			obj := v.Object
			for obj.Next() {
			}
		}
		return r.err
	}
}

func (r *Reader) SetNumberRawRead(readRaw bool) {
	r.tr.options.readRawNumbers = readRaw
}

func (r *Reader) IsPreProcessed() bool {
	return r.tr.options.lazyRead && r.tr.structBuffer.HasNext()
}

func (r *Reader) IsNumbersRaw() bool {
	return r.tr.options.lazyRead && !r.tr.options.computeNumber
}

func (r *Reader) SyncWithPreProcess() {
	if r.tr.structBuffer.Values == nil {
		return
	}
	if r.tr.options.lazyRead {
		r.tr.options.lazyRead = false
		bufferSize := len(*r.tr.structBuffer.Values)
		if bufferSize != 0 {
			lastStruct := (*r.tr.structBuffer.Values)[0]
			r.tr.pos = lastStruct.End
		}
	}
}

func (r *Reader) PreProcess() {
	if r.tr.structBuffer.Values == nil || r.tr.charBuffer == nil {
		return
	}
	r.tr.options.lazyParse = true
	r.tr.options.lazyRead = false
	cr := *r
	*r.tr.structBuffer.Values = (*r.tr.structBuffer.Values)[:0]
	*r.tr.charBuffer = (*r.tr.charBuffer)[:0]
	if r.tr.options.computeString {
		*r.tr.computedValuesBuffer.StringValues = (*r.tr.computedValuesBuffer.StringValues)[:0]
	}
	if r.tr.options.computeNumber {
		*r.tr.computedValuesBuffer.NumberValues = (*r.tr.computedValuesBuffer.NumberValues)[:0]
	}
	r.tr.structBuffer.Pos = 0
	cr.preProcess()
	r.tr.options.lazyRead = true
	r.tr.options.lazyParse = false
}

func (r *Reader) preProcess() {
	value := r.Any()

	if value == nil {
		r.err = fmt.Errorf("can't parse value")
		return
	}

	tree := r.tr.structBuffer.Values

	pos := len(*tree)
	*tree = append(*tree, JsonTreeStruct{Start: r.tr.lastPos, SubTreeSize: 1})

	switch value.Kind {
	case NumberValue:
		if r.tr.options.computeNumber {
			(*tree)[pos].ComputedValueType = NumberComputed
			(*tree)[pos].ComputedValueIndex = len(*r.tr.computedValuesBuffer.NumberValues) - 1
		}
	case StringValue:
		if r.tr.options.computeString {
			(*tree)[pos].ComputedValueType = StringComputed
			(*tree)[pos].ComputedValueIndex = len(*r.tr.computedValuesBuffer.StringValues) - 1
		}
	case ObjectValue:
		for kv := value.Object; kv.Next(); {
			nextPos := len(*tree)
			key := kv.Name()
			r.preProcess()
			if len(*tree) > nextPos {
				(*tree)[pos].SubTreeSize += (*tree)[nextPos].SubTreeSize
				(*tree)[nextPos].AssocValue = key
			}
		}
	case ArrayValue:
		for v := value.Array; v.Next(); {
			nextPos := len(*tree)
			r.preProcess()
			if len(*tree) > nextPos {
				(*tree)[pos].SubTreeSize += (*tree)[nextPos].SubTreeSize
			}
		}
	}
	(*tree)[pos].End = r.tr.pos
}

func typeErrorForNullableValue(err error) error {
	if err != nil {
		switch e := err.(type) { //nolint:gocritic
		case TypeError:
			e.Nullable = true
			return e
		}
	}
	return err
}

func (r *Reader) typeErrorForCurrentToken(expected ValueKind, nullable bool) error {
	v, err := r.tr.Any()
	if err != nil {
		return err
	}
	return TypeError{Expected: expected, Actual: v.Kind, Offset: r.tr.LastPos(), Nullable: nullable}
}

type JsonStructPointer struct {
	Pos    int
	Values *[]JsonTreeStruct
}

type JsonComputedValueType int32

const (
	NothingComputed JsonComputedValueType = iota
	NumberComputed
	StringComputed
)

type JsonComputedValues struct {
	NumberValues *[]NumberProps
	StringValues *[][]byte
}

func (jPointer *JsonStructPointer) HasNext() bool {
	return jPointer.Pos < len(*jPointer.Values)
}

func (jPointer *JsonStructPointer) Next() bool {
	if !jPointer.HasNext() {
		return false
	}
	jPointer.Pos++
	return true
}

func (jPointer *JsonStructPointer) SkipSubTree() bool {
	if jPointer.Pos >= len(*jPointer.Values) {
		return false
	}
	jPointer.Pos += (*jPointer.Values)[jPointer.Pos].SubTreeSize
	return true
}

func (jPointer *JsonStructPointer) CurrentStruct() (JsonTreeStruct, error) {
	if jPointer.Pos >= len(*jPointer.Values) {
		return JsonTreeStruct{}, fmt.Errorf("no elements in structure")
	}
	return (*jPointer.Values)[jPointer.Pos], nil
}

func (jPointer *JsonStructPointer) ReturnBackOn(shift int) bool {
	jPointer.Pos -= shift
	if jPointer.Pos < 0 || jPointer.Pos >= len(*jPointer.Values) {
		jPointer.Pos += shift
		return false
	}
	return true
}

type JsonTreeStruct struct {
	Start              int
	End                int
	SubTreeSize        int
	AssocValue         []byte // for key:value it is key, else nil
	ComputedValueType  JsonComputedValueType
	ComputedValueIndex int
}
