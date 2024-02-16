package jreader

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strconv"
	"strings"
	"testing"
)

func TestRawNumberReading(t *testing.T) {
	type testDef struct {
		prefix string
		suffix string
		input  string
	}

	tests := []testDef{
		{input: "0123456789eE+-."},
		{input: "0123456789eE+-.", prefix: "   ", suffix: "    "},
		{input: "0123456789eE+-.", prefix: "   ", suffix: ",    "},
		{input: "0123456789eE+-.", prefix: "   ", suffix: "  }  "},
		{input: "+11010101010", prefix: "   ", suffix: " ]   "},
	}

	structBuffer := make([]JsonTreeStruct, 0)
	charBuffer := make([]byte, 0)
	buffer := make([]NumberProps, 0)

	for _, test := range tests {
		t.Run(fmt.Sprintf("Test[\"%s\"]", test.input), func(st *testing.T) {
			r := NewReaderWithBuffers(
				[]byte(test.prefix+test.input+test.suffix),
				BufferConfig{
					&structBuffer,
					&charBuffer,
					JsonComputedValues{
						NumberValues: &buffer,
					},
				},
			)
			r.PreProcess()
			result := r.Number()
			ok := r.Error() == nil
			if ok {
				assert.Equal(st, string(result), test.input)
			} else {
				st.Fail()
			}
		})
	}
}

func TestInt64WithComputeWithBuffers(t *testing.T) {
	type testDef struct {
		input   string
		success bool
		result  int64
	}

	tests := []testDef{
		{"0", true, 0},
		{"1", true, 1},
		{"  123123213213  ", true, 123123213213},
		{"-123123213213", true, -123123213213},
		{"  9223372036854775807", true, 9223372036854775807},
		{"-9223372036854775808  ", true, -9223372036854775808},
		{"9223372036854775808", false, 0},
		{"-9223372036854775809", false, 0},
		{" -0.1", false, 0},
		{"+0.1  ", false, 0},
	}

	structBuffer := make([]JsonTreeStruct, 0)
	charBuffer := make([]byte, 0)
	buffer := make([]NumberProps, 0)

	for _, test := range tests {
		t.Run(fmt.Sprintf("Parse int64 (input: %s)", test.input), func(st *testing.T) {
			r := NewReaderWithBuffers(
				[]byte(test.input),
				BufferConfig{
					&structBuffer,
					&charBuffer,
					JsonComputedValues{
						NumberValues: &buffer,
					},
				},
			)
			r.SetNumberRawRead(false)
			r.PreProcess()
			result, ok := r.Int64OrNull()
			if test.success {
				assert.Equal(st, result, test.result)
			} else {
				assert.False(st, ok)
			}
		})
	}
}

func TestInt64WithComputeWithoutBuffers(t *testing.T) {
	type testDef struct {
		input   string
		success bool
		result  int64
	}

	tests := []testDef{
		{"0", true, 0},
		{"1", true, 1},
		{"123123213213  ", true, 123123213213},
		{"  -123123213213  ", true, -123123213213},
		{"9223372036854775807", true, 9223372036854775807},
		{"-9223372036854775808", true, -9223372036854775808},
		{"9223372036854775808", false, 0},
		{"-9223372036854775809", false, 0},
		{"-0.1", false, 0},
		{"+0.1", false, 0},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Parse int64 (input: %s)", test.input), func(st *testing.T) {
			r := NewReader([]byte(test.input))
			r.SetNumberRawRead(false)
			r.PreProcess()
			result, ok := r.Int64OrNull()
			if test.success {
				assert.Equal(st, result, test.result)
			} else {
				assert.False(st, ok)
			}
		})
	}
}

func TestUInt64WithComputeWithoutBuffers(t *testing.T) {
	type testDef struct {
		input   string
		success bool
		result  uint64
	}

	tests := []testDef{
		{"0", true, 0},
		{"1", true, 1},
		{"123123213213  ", true, 123123213213},
		{"  -123123213213  ", false, 0},
		{"9223372036854775807", true, 9223372036854775807},
		{"-9223372036854775808", false, 0},
		{"18446744073709551615", true, 18446744073709551615},
		{"-0.1", false, 0},
		{"+0.1", false, 0},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Parse int64 (input: %s)", test.input), func(st *testing.T) {
			r := NewReader([]byte(test.input))
			r.SetNumberRawRead(false)
			result, ok := r.UInt64OrNull()
			if test.success {
				assert.Equal(st, result, test.result)
			} else {
				assert.False(st, ok)
			}
		})
	}
}

func AtofSuccess(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func Atof(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func TestFloat64WithComputeWithoutBuffers(t *testing.T) {
	type testDef struct {
		input          string
		worksDifferent bool
	}

	tests := []testDef{
		{input: "0"},
		{input: "1"},
		{input: "123123213213  "},
		{input: "  -123123213213  "},
		{input: "9223372036854775807"},
		{input: "-9223372036854775808"},
		{input: "18446744073709551615"},
		{input: "-0.1"},
		{input: "-000000000000.1", worksDifferent: true},
		{input: "0.000000000000000000000000000000000001"},
		{input: "0.123456789111315171921232527"},
		{input: "+0.1", worksDifferent: true},
		{input: "-0.1e3"},
		{input: "-1121211.1e+3"},
		{input: "-1121211.1e+3"},
		{input: "-0.1E-3"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Parse int64 (input: %s)", test.input), func(st *testing.T) {
			r := NewReader([]byte(test.input))
			result, ok := r.Float64OrNull()
			if !test.worksDifferent {
				trimmedValue := strings.TrimSpace(test.input)
				if AtofSuccess(trimmedValue) {
					assert.Equal(st, Atof(trimmedValue), result)
				} else {
					assert.False(st, ok)
				}
			} else {
				assert.False(st, ok)
			}
		})
	}
}

func TestParseCharactersToNumberProperties(t *testing.T) {
	type testDef struct {
		name  string
		input string
		same  bool
	}

	tests := []testDef{
		{"simple int 1", "0 ", true},
		{"simple int 2", "1", true},
		{"simple int 3", "-2", true},
		{"simple int 4", "1234", true},
		{"simple non-json int 1", "001", false},
		{"simple non-json int 2", "-001", false},
		{"simple float 1", "1.2", true},
		{"simple float 2", "-1.2", true},
		{"simple float 3", "1221.212", true},
		{"simple float 4", "0.0002  ", true},
		{"simple trunc float 1", "11111111111111111112", true},
		{"simple trunc float 2", "1111111111.1111111112", true},
		{"simple trunc float 3", "0.0000000000000000001", true},
		{"simple trunc float 4", "0.10000000000000000001", true},
		{"simple trunc float 4", "0.10000000000000000000", true},
		{"simple trunc float 5", "-0.0000000000000000123456789111315171921", true},
		{"simple exp float 1", "1e3", true},
		{"simple exp float 2", "1e19", true},
		{"simple exp float 3", "-234e19", true},
		{"simple exp float 4", "-234e308", true},
		{"simple exp float 5", "-234e11111", true},
		{"simple exp float 6", "-0.0000000000000000001e11111", true},
		{"simple exp float 7", "-111111111111111112345e11111", true},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s (input: %s)", test.name, test.input), func(st *testing.T) {
			BaseTest(st, test.input, test.same)
		})
	}
}

func BaseTest(t *testing.T, s string, expectSame bool) {
	mantissa, exp, neg, trunc, _, ok := runReader(s)
	mantissa2, exp2, neg2, trunc2, _, _, ok2 := originalReadFloat(s)

	fmt.Println(mantissa, exp, neg, trunc, ok)
	fmt.Println(mantissa2, exp2, neg2, trunc2, ok2)

	if expectSame {
		assert.Equal(t, mantissa, mantissa2, "Must be equal mantissas")
		assert.Equal(t, exp, exp2, "Must be equal exponents")
		assert.Equal(t, neg, neg2, "Must be equal sign")
		assert.Equal(t, trunc, trunc2, "Must be equal trunc")
		assert.Equal(t, ok, ok2, "Must be equal result succession")
	} else {
		assert.NotEqual(t, ok, ok2, "Results must be different")
	}
}

func runReader(s string) (mantissa uint64, exp int, neg, trunc bool, bs []byte, ok bool) {
	r := newTokenReader(
		[]byte(s),
		nil,
		nil,
		JsonComputedValues{
			nil,
			nil,
		},
	)
	ch, _ := r.readByte()
	var result NumberProps
	ok = r.readNumberProps(ch, &result)
	if !ok {
		return
	}

	mantissa, exp, neg, trunc, bs = result.mantissa, result.exponent, result.isNegative, result.trunc, result.raw
	return
}

// original function
func originalReadFloat(s string) (mantissa uint64, exp int, neg, trunc, hex bool, i int, ok bool) {
	underscores := false

	// optional sign
	if i >= len(s) {
		return
	}
	switch {
	case s[i] == '+':
		i++
	case s[i] == '-':
		neg = true
		i++
	}

	// digits
	base := uint64(10)
	maxMantDigits := 19 // 10^19 fits in uint64
	expChar := byte('e')
	if i+2 < len(s) && s[i] == '0' && lower(s[i+1]) == 'x' {
		base = 16
		maxMantDigits = 16 // 16^16 fits in uint64
		i += 2
		expChar = 'p'
		hex = true
	}
	sawdot := false
	sawdigits := false
	nd := 0
	ndMant := 0
	dp := 0
loop:
	for ; i < len(s); i++ {
		switch c := s[i]; true {
		case c == '_':
			underscores = true
			continue

		case c == '.':
			if sawdot {
				break loop
			}
			sawdot = true
			dp = nd
			continue

		case '0' <= c && c <= '9':
			sawdigits = true
			if c == '0' && nd == 0 { // ignore leading zeros
				dp--
				continue
			}
			nd++
			if ndMant < maxMantDigits {
				mantissa *= base
				mantissa += uint64(c - '0')
				ndMant++
			} else if c != '0' {
				trunc = true
			}
			continue

		case base == 16 && 'a' <= lower(c) && lower(c) <= 'f':
			sawdigits = true
			nd++
			if ndMant < maxMantDigits {
				mantissa *= 16
				mantissa += uint64(lower(c) - 'a' + 10)
				ndMant++
			} else {
				trunc = true
			}
			continue
		}
		break
	}
	if !sawdigits {
		return
	}
	if !sawdot {
		dp = nd
	}

	if base == 16 {
		dp *= 4
		ndMant *= 4
	}

	// optional exponent moves decimal point.
	// if we read a very large, very long number,
	// just be sure to move the decimal point by
	// a lot (say, 100000).  it doesn't matter if it's
	// not the exact number.
	if i < len(s) && lower(s[i]) == expChar {
		i++
		if i >= len(s) {
			return
		}
		esign := 1
		if s[i] == '+' {
			i++
		} else if s[i] == '-' {
			i++
			esign = -1
		}
		if i >= len(s) || s[i] < '0' || s[i] > '9' {
			return
		}
		e := 0
		for ; i < len(s) && ('0' <= s[i] && s[i] <= '9' || s[i] == '_'); i++ {
			if s[i] == '_' {
				underscores = true
				continue
			}
			if e < 10000 {
				e = e*10 + int(s[i]) - '0'
			}
		}
		dp += e * esign
	} else if base == 16 {
		// Must have exponent.
		return
	}

	if mantissa != 0 {
		exp = dp - ndMant
	}

	if underscores && !underscoreOK(s[:i]) {
		return
	}

	ok = true
	return
}

func lower(c byte) byte {
	return c | ('x' - 'X')
}

func underscoreOK(s string) bool {
	// saw tracks the last character (class) we saw:
	// ^ for beginning of number,
	// 0 for a digit or base prefix,
	// _ for an underscore,
	// ! for none of the above.
	saw := '^'
	i := 0

	// Optional sign.
	if len(s) >= 1 && (s[0] == '-' || s[0] == '+') {
		s = s[1:]
	}

	// Optional base prefix.
	hex := false
	if len(s) >= 2 && s[0] == '0' && (lower(s[1]) == 'b' || lower(s[1]) == 'o' || lower(s[1]) == 'x') {
		i = 2
		saw = '0' // base prefix counts as a digit for "underscore as digit separator"
		hex = lower(s[1]) == 'x'
	}

	// Number proper.
	for ; i < len(s); i++ {
		// Digits are always okay.
		if '0' <= s[i] && s[i] <= '9' || hex && 'a' <= lower(s[i]) && lower(s[i]) <= 'f' {
			saw = '0'
			continue
		}
		// Underscore must follow digit.
		if s[i] == '_' {
			if saw != '0' {
				return false
			}
			saw = '_'
			continue
		}
		// Underscore must also be followed by digit.
		if saw == '_' {
			return false
		}
		// Saw non-digit, non-underscore.
		saw = '!'
	}
	return saw != '_'
}
