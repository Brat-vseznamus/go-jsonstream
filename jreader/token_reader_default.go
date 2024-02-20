//go:build !launchdarkly_easyjson
// +build !launchdarkly_easyjson

package jreader

// This file defines the default implementation of the low-level JSON tokenizer. If the launchdarkly_easyjson
// build tag is enabled, we use the easyjson adapter in token_reader_easyjson.go instead. These have the same
// methods so the Reader code does not need to know which implementation we're using; however, we don't
// actually define an interface for these, because calling the methods through an interface would limit
// performance.

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"unicode"
	"unicode/utf8"
)

var (
	tokenNull  = []byte("null")  //nolint:gochecknoglobals
	tokenTrue  = []byte("true")  //nolint:gochecknoglobals
	tokenFalse = []byte("false") //nolint:gochecknoglobals
)

type token struct {
	kind        tokenKind
	boolValue   bool
	numberValue NumberProps
	stringValue []byte
	delimiter   byte
}

type tokenKind int

const (
	nullToken      tokenKind = iota
	boolToken      tokenKind = iota
	numberToken    tokenKind = iota
	stringToken    tokenKind = iota
	delimiterToken tokenKind = iota
)

func (t token) valueKind() ValueKind {
	if t.kind == delimiterToken {
		if t.delimiter == '[' {
			return ArrayValue
		}
		if t.delimiter == '{' {
			return ObjectValue
		}
	}
	return valueKindFromTokenKind(t.kind)
}

func (t token) description() string {
	if t.kind == delimiterToken && t.delimiter != '[' && t.delimiter != '{' {
		return "'" + string(t.delimiter) + "'"
	}
	return t.valueKind().String()
}

type readerOptions struct {
	lazyParse      bool
	lazyRead       bool
	computeString  bool
	computeNumber  bool // TODO
	readKey        bool
	readRawNumbers bool
}

type tokenReader struct {
	data                 []byte
	pos                  int
	len                  int
	hasUnread            bool
	unreadToken          token
	lastPos              int
	charBuffer           *[]byte
	structBuffer         JsonStructPointer
	computedValuesBuffer JsonComputedValues
	anyValueBuffer       AnyValue
	tokenBuffer          token
	options              readerOptions
}

func newTokenReader(data []byte, buffer *[]JsonTreeStruct, charBuffer *[]byte, computedValuesBuffer JsonComputedValues) tokenReader {
	tr := tokenReader{
		structBuffer: JsonStructPointer{
			Values: buffer,
		},
		charBuffer:           charBuffer,
		computedValuesBuffer: computedValuesBuffer,
	}
	tr.Reset(data)
	return tr
}

func (r *tokenReader) Reset(data []byte) {
	r.data = data
	r.len = len(data)
	r.pos = 0
	r.hasUnread = false

	if r.charBuffer != nil {
		*r.charBuffer = (*r.charBuffer)[:0]
	}
	r.structBuffer.Pos = 0
	if r.structBuffer.Values != nil {
		*r.structBuffer.Values = (*r.structBuffer.Values)[:0]
	}
	r.options.computeString = r.computedValuesBuffer.StringValues != nil
	if r.options.computeString {
		*r.computedValuesBuffer.StringValues = (*r.computedValuesBuffer.StringValues)[:0]
	}
	r.options.computeNumber = r.computedValuesBuffer.NumberValues != nil
	if r.options.computeNumber {
		*r.computedValuesBuffer.NumberValues = (*r.computedValuesBuffer.NumberValues)[:0]
	}
	r.options.readKey = false
	r.options.lazyParse = false
	r.options.lazyRead = false
	r.options.readRawNumbers = true
}

// EOF returns true if we are at the end of the input (not counting whitespace).
func (r *tokenReader) EOF() bool {
	if r.hasUnread {
		return false
	}
	_, ok := r.skipWhitespaceAndReadByte()
	if !ok {
		return true
	}
	r.unreadByte()
	return false
}

// LastPos returns the byte offset within the input where we most recently started parsing a token.
func (r *tokenReader) LastPos() int {
	return r.lastPos
}

func (r *tokenReader) getPos() int {
	if r.hasUnread {
		return r.lastPos
	}
	return r.pos
}

// Null returns (true, nil) if the next token is a null (consuming the token); (false, nil) if the next
// token is not a null (not consuming the token); or (false, error) if the next token is not a valid
// JSON Value.
//
// This and all other tokenReader methods skip transparently past whitespace between tokens.
func (r *tokenReader) Null() (bool, error) {
	t, err := r.next()
	if t == nil {
		return false, err
	}
	if err != nil {
		return false, err
	}
	if t.kind == nullToken {
		return true, nil
	}
	r.putBack(t)
	if t.kind == delimiterToken && t.delimiter != '[' && t.delimiter != '{' {
		return false, SyntaxError{Message: errMsgUnexpectedChar, Value: string(t.delimiter), Offset: r.getPos()}
	}
	return false, nil
}

// Bool requires that the next token is a JSON boolean, returning its Value if successful (consuming
// the token), or an error if the next token is anything other than a JSON boolean.
//
// This and all other tokenReader methods skip transparently past whitespace between tokens.
func (r *tokenReader) Bool() (bool, error) {
	t, err := r.consumeScalar(boolToken)
	if t == nil {
		return false, err
	}
	return t.boolValue, err
}

// Number requires that the next token is a JSON number, returning its Value if successful (consuming
// the token), or an error if the next token is anything other than a JSON number.
//
// This and all other tokenReader methods skip transparently past whitespace between tokens.
func (r *tokenReader) Number() (*NumberProps, error) {
	t, err := r.consumeScalar(numberToken)
	if t == nil {
		return nil, err
	}
	return &t.numberValue, err
}

// String requires that the next token is a JSON string, returning its Value if successful (consuming
// the token), or an error if the next token is anything other than a JSON string.
//
// This and all other tokenReader methods skip transparently past whitespace between tokens.
func (r *tokenReader) String() ([]byte, error) {
	t, err := r.consumeScalar(stringToken)
	if t == nil {
		return nil, err
	}
	return t.stringValue, err
}

// PropertyName requires that the next token is a JSON string and the token after that is a colon,
// returning the string as a byte slice if successful, or an error otherwise.
//
// Returning the string as a byte slice avoids the overhead of allocating a string, since normally
// the names of properties will not be retained as strings but are only compared to constants while
// parsing an object.
//
// This and all other tokenReader methods skip transparently past whitespace between tokens.
func (r *tokenReader) PropertyName() ([]byte, error) {
	r.options.readKey = true
	defer func() {
		r.options.readKey = false
	}()
	t, err := r.consumeScalar(stringToken)
	if t == nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	b, ok := r.skipWhitespaceAndReadByte()
	if !ok {
		return nil, io.EOF
	}
	if b != ':' {
		r.unreadByte()
		return nil, r.syntaxErrorOnNextToken(errMsgExpectedColon)
	}
	return t.stringValue, nil
}

// Delimiter checks whether the next token is the specified ASCII delimiter character. If so, it
// returns (true, nil) and consumes the token. If it is a delimiter, but not the same one, it
// returns (false, nil) and does not consume the token. For anything else, it returns an error.
//
// This and all other tokenReader methods skip transparently past whitespace between tokens.
func (r *tokenReader) Delimiter(delimiter byte) (bool, error) {
	if r.options.lazyRead {
		currStruct, err := r.structBuffer.CurrentStruct()
		if err != nil {
			return false, err
		}
		return r.data[currStruct.Start] == delimiter, nil
	} else {
		if r.hasUnread {
			if r.unreadToken.kind == delimiterToken && r.unreadToken.delimiter == delimiter {
				r.hasUnread = false
				return true, nil
			}
			return false, nil
		}
		b, ok := r.skipWhitespaceAndReadByte()
		if !ok {
			return false, nil
		}
		if b == delimiter {
			return true, nil
		}
		r.unreadByte() // we'll back up and try to parse a token, to see if it's valid JSON or not
		token, err := r.next()
		if token == nil {
			return false, err
		}
		if err != nil {
			return false, err // it was malformed JSON
		}
		r.putBack(token) // it was valid JSON, we just haven't hit that delimiter
		return false, nil
	}
}

// EndDelimiterOrComma checks whether the next token is the specified ASCII delimiter character
// or a comma. If it is the specified delimiter, it returns (true, nil) and consumes the token.
// If it is a comma, it returns (false, nil) and consumes the token. For anything else, it
// returns an error. The delimiter parameter will always be either '}' or ']'.
func (r *tokenReader) EndDelimiterOrComma(delimiter byte) (bool, error) {
	if r.options.lazyRead {
		return false, fmt.Errorf("can't be used in lazy mode")
	} else {
		if r.hasUnread {
			if r.unreadToken.kind == delimiterToken &&
				(r.unreadToken.delimiter == delimiter || r.unreadToken.delimiter == ',') {
				r.hasUnread = false
				return r.unreadToken.delimiter == delimiter, nil
			}
			return false, SyntaxError{Message: badArrayOrObjectItemMessage(delimiter == '}'),
				Value: r.unreadToken.description(), Offset: r.lastPos}
		}
		b, ok := r.skipWhitespaceAndReadByte()
		if !ok {
			return false, io.EOF
		}
		if b == delimiter || b == ',' {
			return b == delimiter, nil
		}
		r.unreadByte()
		t, err := r.next()
		if t == nil {
			return false, err
		}
		if err != nil {
			return false, err
		}
		return false, SyntaxError{Message: badArrayOrObjectItemMessage(delimiter == '}'),
			Value: t.description(), Offset: r.lastPos}
	}
}

func badArrayOrObjectItemMessage(isObject bool) string {
	if isObject {
		return errMsgBadObjectItem
	}
	return errMsgBadArrayItem
}

// Any checks whether the next token is either a valid JSON scalar Value or the opening delimiter of
// an array or object Value. If so, it returns (AnyValue, nil) and consumes the token; if not, it
// returns an error. Unlike Reader.Any(), for array and object values it does not create an
// ArrayState or ObjectState.
func (r *tokenReader) Any() (*AnyValue, error) {
	t, err := r.next()
	if t == nil {
		return nil, err
	}
	if err != nil {
		return &r.anyValueBuffer, err
	}
	switch t.kind {
	case boolToken:
		r.anyValueBuffer.Kind = BoolValue
		r.anyValueBuffer.Bool = t.boolValue
		return &r.anyValueBuffer, nil
	case numberToken:
		r.anyValueBuffer.Kind = NumberValue
		r.anyValueBuffer.Number = t.numberValue
		return &r.anyValueBuffer, nil
	case stringToken:
		r.anyValueBuffer.Kind = StringValue
		r.anyValueBuffer.String = t.stringValue
		return &r.anyValueBuffer, nil
	case delimiterToken:
		if t.delimiter == '[' {
			r.anyValueBuffer.Kind = ArrayValue
			return &r.anyValueBuffer, nil
		}
		if t.delimiter == '{' {
			r.anyValueBuffer.Kind = ObjectValue
			return &r.anyValueBuffer, nil
		}
		return nil,
			SyntaxError{Message: errMsgUnexpectedChar, Value: string(t.delimiter), Offset: r.lastPos}
	default:
		r.anyValueBuffer.Kind = NullValue
		return &r.anyValueBuffer, nil
	}
}

// Attempts to parse and consume the next token, ignoring whitespace. A token is either a valid JSON scalar
// Value or an ASCII delimiter character. If a token was previously unread using putBack, it consumes that
// instead.
func (r *tokenReader) next() (*token, error) {
	if r.hasUnread {
		r.hasUnread = false
		return &r.unreadToken, nil
	}
	b, ok := r.skipWhitespaceAndReadByte()
	if !ok {
		return nil, io.EOF
	}

	switch {
	// We can get away with reading bytes instead of runes because the JSON spec doesn't allow multi-byte
	// characters except within a string literal.
	case b >= 'a' && b <= 'z':
		if r.options.lazyRead {
			r.structBuffer.Next()
			if b == 'f' {
				r.tokenBuffer.kind = boolToken
				r.tokenBuffer.boolValue = false
				return &r.tokenBuffer, nil
			}
			if b == 't' {
				r.tokenBuffer.kind = boolToken
				r.tokenBuffer.boolValue = true
				return &r.tokenBuffer, nil
			}
			if b == 'n' {
				r.tokenBuffer.kind = nullToken
				return &r.tokenBuffer, nil
			}
			return nil, SyntaxError{Message: errMsgUnexpectedSymbol, Value: string(b), Offset: r.lastPos}
		} else {
			n := r.consumeASCIILowercaseAlphabeticChars() + 1
			id := r.data[r.lastPos : r.lastPos+n]
			if b == 'f' && bytes.Equal(id, tokenFalse) {
				r.tokenBuffer.kind = boolToken
				r.tokenBuffer.boolValue = false
				return &r.tokenBuffer, nil
			}
			if b == 't' && bytes.Equal(id, tokenTrue) {
				r.tokenBuffer.kind = boolToken
				r.tokenBuffer.boolValue = true
				return &r.tokenBuffer, nil
			}
			if b == 'n' && bytes.Equal(id, tokenNull) {
				r.tokenBuffer.kind = nullToken
				return &r.tokenBuffer, nil
			}
			return nil, SyntaxError{Message: errMsgUnexpectedSymbol, Value: string(id), Offset: r.lastPos}
		}
	case (b >= '0' && b <= '9') || b == '-':
		if r.options.lazyRead {
			curStruct, _ := r.structBuffer.CurrentStruct()
			if r.options.computeNumber {
				r.tokenBuffer.numberValue = (*r.computedValuesBuffer.NumberValues)[curStruct.ComputedValueIndex]
			} else {
				nBytes := r.data[curStruct.Start:curStruct.End]
				r.tokenBuffer.numberValue = NumberProps{raw: nBytes}
			}
			r.structBuffer.Next()
			r.tokenBuffer.kind = numberToken
			return &r.tokenBuffer, nil
		} else {
			if n, ok := r.readNumber(b); ok {
				r.tokenBuffer.kind = numberToken
				r.tokenBuffer.numberValue = n
				return &r.tokenBuffer, nil
			}
			return nil, SyntaxError{Message: errMsgInvalidNumber, Offset: r.lastPos}
		}
	case b == '"':
		if r.options.lazyRead {
			curStruct, _ := r.structBuffer.CurrentStruct()
			sBytes := r.data[(curStruct.Start + 1):(curStruct.End - 1)]
			if r.options.computeString && !r.options.readKey {
				sBytes = (*r.computedValuesBuffer.StringValues)[curStruct.ComputedValueIndex]
			}
			r.structBuffer.Next()
			r.tokenBuffer.kind = stringToken
			r.tokenBuffer.stringValue = sBytes
			return &r.tokenBuffer, nil
		} else {
			s, err := r.readString()
			if err != nil {
				return nil, err
			}
			r.tokenBuffer.kind = stringToken
			r.tokenBuffer.stringValue = s
			return &r.tokenBuffer, nil
		}
	case b == '[', b == ']', b == '{', b == '}', b == ':', b == ',':
		r.tokenBuffer.kind = delimiterToken
		r.tokenBuffer.delimiter = b
		return &r.tokenBuffer, nil
	}

	return nil, SyntaxError{Message: errMsgUnexpectedChar, Value: string(b), Offset: r.lastPos}
}

func (r *tokenReader) putBack(token *token) {
	r.unreadToken = *token
	r.hasUnread = true
}

func (r *tokenReader) consumeScalar(kind tokenKind) (*token, error) {
	t, err := r.next()
	if err != nil {
		return nil, err
	}
	if t.kind == kind {
		return t, nil
	}
	if t.kind == delimiterToken && t.delimiter != '[' && t.delimiter != '{' {
		return nil, SyntaxError{Message: errMsgUnexpectedChar, Value: string(t.delimiter), Offset: r.LastPos()}
	}
	return nil, TypeError{Expected: valueKindFromTokenKind(kind),
		Actual: t.valueKind(), Offset: r.LastPos()}
}

func (r *tokenReader) readByte() (byte, bool) {
	if r.pos >= r.len {
		return 0, false
	}
	b := r.data[r.pos]
	r.pos++
	return b, true
}

func (r *tokenReader) unreadByte() {
	r.pos--
}

func (r *tokenReader) skipWhitespaceAndReadByte() (byte, bool) {
	if r.options.lazyRead {
		curStruct, err := r.structBuffer.CurrentStruct()
		if err != nil {
			return 0, false
		}
		return r.data[curStruct.Start], true
	} else {
		for {
			ch, ok := r.readByte()
			if !ok {
				return 0, false
			}
			if !unicode.IsSpace(rune(ch)) {
				r.lastPos = r.pos - 1
				return ch, true
			}
		}
	}
}

func (r *tokenReader) consumeASCIILowercaseAlphabeticChars() int {
	n := 0
	for {
		ch, ok := r.readByte()
		if !ok {
			break
		}
		if ch < 'a' || ch > 'z' {
			r.unreadByte()
			break
		}
		n++
	}
	return n
}

func (r *tokenReader) readNumber(first byte) (result NumberProps, ok bool) { //nolint:unparam
	ok = r.readNumberProps(first, &result)
	if ok && r.options.lazyParse && r.options.computeNumber {
		nValues := r.computedValuesBuffer.NumberValues
		*nValues = append(*nValues, result)
	}
	return
}

func (r *tokenReader) readString() ([]byte, error) {
	startPos := r.pos
	chars := r.charBuffer
	charsStartPos := len(*chars)

	haveEscaped := false
	var reader bytes.Reader // bytes.Reader understands multi-byte characters
	reader.Reset(r.data)
	_, _ = reader.Seek(int64(r.pos), io.SeekStart)

	for {
		ch, _, err := reader.ReadRune()
		if err != nil {
			return nil, r.syntaxErrorOnLastToken(errMsgInvalidString)
		}
		if r.options.readKey || !r.options.computeString {
			if ch == '\\' {
				haveEscaped = !haveEscaped
			} else if ch == '"' && !haveEscaped {
				break
			} else {
				haveEscaped = false
			}
		} else {
			if ch == '"' {
				break
			}
			if ch != '\\' {
				*chars = appendRune(*chars, ch)
				continue
			}
			ch, _, err = reader.ReadRune()
			if err != nil {
				return nil, r.syntaxErrorOnLastToken(errMsgInvalidString)
			}
			switch ch {
			case '"', '\\', '/':
				*chars = appendRune(*chars, ch)
			case 'b':
				*chars = appendRune(*chars, '\b')
			case 'f':
				*chars = appendRune(*chars, '\f')
			case 'n':
				*chars = appendRune(*chars, '\n')
			case 'r':
				*chars = appendRune(*chars, '\r')
			case 't':
				*chars = appendRune(*chars, '\t')
			case 'u':
				if ch, ok := readHexChar(&reader); ok {
					*chars = appendRune(*chars, ch)
				} else {
					return nil, r.syntaxErrorOnLastToken(errMsgInvalidString)
				}
			default:
				return nil, r.syntaxErrorOnLastToken(errMsgInvalidString)
			}
		}
	}
	r.pos = r.len - reader.Len()

	if r.options.readKey || !r.options.computeString {
		pos := r.pos - 1
		if pos <= startPos {
			return nil, nil
		}
		return r.data[startPos:pos], nil
	} else {
		charsEndPos := len(*chars)
		if r.options.lazyParse {
			sValues := r.computedValuesBuffer.StringValues
			*sValues = append(*sValues, (*chars)[charsStartPos:charsEndPos])
		}
		if charsEndPos == charsStartPos {
			return nil, nil
		}
		return (*chars)[charsStartPos:charsEndPos], nil
	}
}

func readHexChar(reader *bytes.Reader) (rune, bool) {
	var digits [4]byte
	for i := 0; i < 4; i++ {
		ch, err := reader.ReadByte()
		if err != nil || !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
			return 0, false
		}
		digits[i] = ch
	}
	n, _ := strconv.ParseUint(string(digits[:]), 16, 32)
	return rune(n), true
}

func (r *tokenReader) syntaxErrorOnLastToken(msg string) error { //nolint:unparam
	return SyntaxError{Message: msg, Offset: r.LastPos()}
}

func (r *tokenReader) syntaxErrorOnNextToken(msg string) error {
	t, err := r.next()
	if err != nil {
		return err
	}
	return SyntaxError{Message: msg, Value: t.description(), Offset: r.LastPos()}
}

func appendRune(out []byte, ch rune) []byte {
	var encodedRune [10]byte
	n := utf8.EncodeRune(encodedRune[0:10], ch)
	return append(out, encodedRune[0:n]...)
}

func valueKindFromTokenKind(k tokenKind) ValueKind {
	switch k {
	case nullToken:
		return NullValue
	case boolToken:
		return BoolValue
	case numberToken:
		return NumberValue
	case stringToken:
		return StringValue
	}
	return -1
}
