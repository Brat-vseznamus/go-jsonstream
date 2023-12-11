package jreader

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

func TestDestructStrings(t *testing.T) {
	buffer := make([]JsonTreeStruct, 0, 100)
	charBuffer := make([]byte, 0, 100)
	stringsBuffer := make([][]byte, 0, 10)

	type TestValuesData struct {
		testName       string
		computeStrings bool
		input          JsonElement
		expectedOutput JsonElement
	}

	values := []TestValuesData{
		{
			testName:       "strings should not computed, and there are no escaping",
			computeStrings: false,
			input:          JsonString("\"abc\""),
			expectedOutput: JsonString("\"abc\""),
		},
		{
			testName:       "strings should not computed, and there are some escaping",
			computeStrings: false,
			input:          JsonString("\"\\nabc\""),
			expectedOutput: JsonString("\"\\nabc\""),
		},
		{
			testName:       "strings must be computed, and there are no escaping",
			computeStrings: true,
			input:          JsonString("\"abc\""),
			expectedOutput: JsonString("\"abc\""),
		},
		{
			testName:       "strings must be computed, and there are some escaping",
			computeStrings: true,
			input:          JsonString("\"\\n\\t\\u00bfabc\""),
			expectedOutput: JsonString("\"\n\t¿abc\""),
		},
	}

	for _, testValuesData := range values {
		obj := testValuesData.input
		objStr := obj.JsonToString()

		r := NewReaderWithBuffers([]byte(objStr), BufferConfig{
			StructBuffer: &buffer,
			CharsBuffer:  &charBuffer,
		})

		if testValuesData.computeStrings {
			r = NewReaderWithBuffers([]byte(objStr), BufferConfig{
				StructBuffer: &buffer,
				CharsBuffer:  &charBuffer,
				ComputedValuesBuffer: JsonComputedValues{
					StringValues: &stringsBuffer,
				},
			})
		}

		r.PreProcess()

		t.Run(testValuesData.testName, func(subT *testing.T) {
			assert.Equal(subT, testValuesData.expectedOutput, Build(&r))
		})
	}
}

func TestDestructAtoms(t *testing.T) {
	buffer := make([]JsonTreeStruct, 0, 100)
	charBuffer := make([]byte, 0, 100)

	values := []JsonPair{
		{
			k: "null",
			v: JsonNull{},
		},
		{
			k: "string",
			v: JsonString("\"string\""),
		},
		{
			k: "number",
			v: JsonNumber("123.4"),
		},
		{
			k: "bool",
			v: JsonBool(true),
		},
	}

	for _, kv := range values {
		obj := kv.v
		objStr := kv.v.JsonToString()

		r := NewReaderWithBuffers([]byte(objStr), BufferConfig{
			StructBuffer: &buffer,
			CharsBuffer:  &charBuffer,
		})

		r.PreProcess()

		t.Run(kv.k, func(subT *testing.T) {
			assert.Equal(subT, obj, Build(&r))
		})
	}
}

func TestDestructArrays(t *testing.T) {
	buffer := make([]JsonTreeStruct, 0, 100)
	charBuffer := make([]byte, 0, 100)

	values := []JsonPair{
		{
			k: "empty array",
			v: JsonArray{},
		},
		{
			k: "single value array",
			v: JsonArray{JsonNumber("123.4")},
		},
		{
			k: "multiple values array",
			v: JsonArray{
				JsonNumber("123.4"),
				JsonString("\"234.5\""),
				JsonNumber("345.6"),
			},
		},
	}

	for _, kv := range values {
		obj := kv.v
		objStr := kv.v.JsonToString()

		r := NewReaderWithBuffers([]byte(objStr), BufferConfig{
			StructBuffer: &buffer,
			CharsBuffer:  &charBuffer,
		})
		r.PreProcess()

		t.Run(kv.k, func(subT *testing.T) {
			assert.Equal(subT, obj, Build(&r))
		})
	}
}

func TestDestructObjects(t *testing.T) {
	buffer := make([]JsonTreeStruct, 0, 100)
	charBuffer := make([]byte, 0, 100)
	//stringsBuffer := make([][]byte, 0, 100)

	values := []JsonPair{
		{
			k: "empty object",
			v: JsonObject{},
		},
		{
			k: "single key object",
			v: JsonObject{
				JsonPair{
					k: "1",
					v: JsonNumber([]byte("123.4")),
				},
			},
		},
		{
			k: "multiple keys object",
			v: JsonObject{
				JsonPair{
					k: "1",
					v: JsonNumber([]byte("123.4")),
				},
				JsonPair{
					k: "2",
					v: JsonNumber([]byte("123.45")),
				},
				JsonPair{
					k: "3",
					v: JsonNumber([]byte("123.456")),
				},
			},
		},
	}

	for _, kv := range values {
		obj := kv.v
		objStr := kv.v.JsonToString()

		r := NewReaderWithBuffers([]byte(objStr), BufferConfig{
			StructBuffer: &buffer,
			CharsBuffer:  &charBuffer,
		})
		r.PreProcess()

		t.Run(kv.k, func(subT *testing.T) {
			buildResult := Build(&r)
			assert.Equal(subT, obj, buildResult)
		})
	}
}

func TestDestructRandom(t *testing.T) {
	buffer := make([]JsonTreeStruct, 0, 100)
	charBuffer := make([]byte, 0, 100)

	sizes := []int{0, 1, 2, 4, 10, 100, 1000, 100000}

	for _, s := range sizes {
		obj := RandomJson(s)
		objStr := obj.JsonToString()

		r := NewReaderWithBuffers([]byte(objStr), BufferConfig{
			StructBuffer: &buffer,
			CharsBuffer:  &charBuffer,
		})
		r.PreProcess()

		t.Run(fmt.Sprintf("json element with volume %d", s), func(subT *testing.T) {
			assert.Equal(subT, obj, Build(&r))
		})
	}
}

func TestPartialDestruct(t *testing.T) {
	buffer := make([]JsonTreeStruct, 0, 100)
	charBuffer := make([]byte, 0, 100)

	obj := JsonObject{
		JsonPair{
			k: "f1",
			v: JsonNumber([]byte("222")),
		},
		JsonPair{
			k: "f2",
			v: JsonObject{},
		},
		JsonPair{
			k: "f3",
			v: JsonArray{
				JsonObject{
					JsonPair{
						k: "f4",
						v: JsonString("\"222\""),
					},
				},
			},
		},
	}
	objStr := obj.JsonToString()

	r := NewReaderWithBuffers([]byte(objStr), BufferConfig{
		StructBuffer: &buffer,
		CharsBuffer:  &charBuffer,
	})

	je := BuildWithPartialDestruct(&r)
	assert.Equal(t, obj, je)
}

func TestPartialDestructRandom(t *testing.T) {
	buffer := make([]JsonTreeStruct, 0, 100)
	charBuffer := make([]byte, 0, 100)

	sizes := []int{0, 1, 2, 4, 10, 100, 1000, 100000}

	for _, s := range sizes {
		obj := RandomJson(s)
		objStr := obj.JsonToString()

		r := NewReaderWithBuffers([]byte(objStr), BufferConfig{
			StructBuffer: &buffer,
			CharsBuffer:  &charBuffer,
		})

		t.Run(fmt.Sprintf("json element with volume %d", s), func(subT *testing.T) {
			assert.Equal(subT, obj, BuildWithPartialDestruct(&r))
		})
	}
}

func BuildWithPartialDestruct(r *Reader) JsonElement {
	value := r.Any()
	switch value.Kind {
	case NumberValue:
		return JsonNumber(value.Number.raw)
	case StringValue:
		return JsonString("\"" + string(value.String) + "\"")
	case BoolValue:
		return JsonBool(value.Bool)
	case NullValue:
		return JsonNull{}
	case ObjectValue:
		jo := JsonObject{}
		for kv := value.Object; kv.Next(); {
			isPreProcessed := r.IsPreProcessed()
			if !isPreProcessed {
				r.PreProcess()
			}
			jo = append(jo, JsonPair{k: string(kv.name), v: BuildWithPartialDestruct(r)})
			if !isPreProcessed {
				r.SyncWithPreProcess()
			}
		}
		return jo
	case ArrayValue:
		ja := JsonArray{}
		for v := value.Array; v.Next(); {
			ja = append(ja, BuildWithPartialDestruct(r))
		}
		return ja
	}
	return JsonNull{}
}

func RandomJson(volume int) JsonElement {
	switch volume {
	case 0:
		switch rand.Int() % 2 {
		case 0:
			return JsonArray{}
		case 1:
			return JsonObject{}
		}
	case 1:
		switch rand.Int() % 4 {
		case 0:
			return JsonBool(rand.Int()%2 == 0)
		case 1:
			return JsonNumber([]byte(fmt.Sprintf("%d", rand.Int()%1000000)))
		case 2:
			return JsonString(fmt.Sprintf("\"s%d\"", rand.Int()%1000000))
		case 3:
			return JsonNull{}
		}
	default:
		switch rand.Int() % 2 {
		case 0:
			ja := JsonArray{}
			for volume > 0 {
				subVolume := rand.Int() % (volume + 1)
				volume -= subVolume
				ja = append(ja, RandomJson(subVolume))
			}
			return ja
		case 1:
			jo := JsonObject{}
			for volume > 0 {
				subVolume := rand.Int() % (volume + 1)
				volume -= subVolume
				jo = append(jo, JsonPair{k: fmt.Sprintf("k%d", rand.Int()%1000), v: RandomJson(subVolume)})
			}
			return jo
		}
	}
	return JsonNull{}
}

func Build(r *Reader) JsonElement {
	value := r.Any()
	switch value.Kind {
	case NumberValue:
		return JsonNumber(value.Number.raw)
	case StringValue:
		return JsonString("\"" + string(value.String) + "\"")
	case BoolValue:
		return JsonBool(value.Bool)
	case NullValue:
		return JsonNull{}
	case ObjectValue:
		jo := JsonObject{}
		for kv := value.Object; kv.Next(); {
			jo = append(jo, JsonPair{k: string(kv.name), v: Build(r)})
		}
		return jo
	case ArrayValue:
		ja := JsonArray{}
		for v := value.Array; v.Next(); {
			ja = append(ja, Build(r))
		}
		return ja
	}
	return JsonNull{}
}

type JsonElement interface {
	JsonToString() string
}

type JsonString string
type JsonNumber []byte
type JsonBool bool
type JsonNull struct{}
type JsonArray []JsonElement

type JsonPair struct {
	k string
	v JsonElement
}

type JsonObject []JsonPair

func (j JsonString) JsonToString() string {
	return string(j)
}

func (j JsonNumber) JsonToString() string {
	return string(j)
}

func (j JsonBool) JsonToString() string {
	return fmt.Sprintf("%t", bool(j))
}

func (j JsonNull) JsonToString() string {
	return "null"
}

func (j JsonArray) JsonToString() string {
	s := "["
	for i, v := range j {
		if i != 0 {
			s += ","
		}
		s += v.JsonToString()
	}
	s += "]"
	return s
}

func (j JsonObject) JsonToString() string {
	s := "{"
	for i, v := range j {
		if i != 0 {
			s += ","
		}
		s += fmt.Sprintf(`"%s": %s`, v.k, v.v.JsonToString())
	}
	s += "}"
	return s
}
