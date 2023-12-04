package jreader

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

type JsonElement interface {
	JsonToString() string
}

type JsonString string
type JsonNumber Number
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
	return string(j.Value)
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
			v: JsonNumber{Value: []byte("123.4")},
		},
		{
			k: "bool",
			v: JsonBool(true),
		},
	}

	for _, kv := range values {
		obj := kv.v
		objStr := kv.v.JsonToString()

		r := NewReaderWithBuffers([]byte(objStr), &buffer, &charBuffer)
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
			v: JsonArray{JsonNumber{Value: []byte("123.4")}},
		},
		{
			k: "multiple values array",
			v: JsonArray{
				JsonNumber{Value: []byte("123.4")},
				JsonString("\"234.5\""),
				JsonNumber{Value: []byte("345.6")},
			},
		},
	}

	for _, kv := range values {
		obj := kv.v
		objStr := kv.v.JsonToString()

		r := NewReaderWithBuffers([]byte(objStr), &buffer, &charBuffer)
		r.PreProcess()

		t.Run(kv.k, func(subT *testing.T) {
			assert.Equal(subT, obj, Build(&r))
		})
	}
}

func TestDestructObjects(t *testing.T) {
	buffer := make([]JsonTreeStruct, 0, 100)
	charBuffer := make([]byte, 0, 100)

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
					v: JsonNumber{Value: []byte("123.4")},
				},
			},
		},
		{
			k: "multiple keys object",
			v: JsonObject{
				JsonPair{
					k: "1",
					v: JsonNumber{Value: []byte("123.4")},
				},
				JsonPair{
					k: "2",
					v: JsonNumber{Value: []byte("123.45")},
				},
				JsonPair{
					k: "3",
					v: JsonNumber{Value: []byte("123.456")},
				},
			},
		},
	}

	for _, kv := range values {
		obj := kv.v
		objStr := kv.v.JsonToString()

		r := NewReaderWithBuffers([]byte(objStr), &buffer, &charBuffer)
		r.PreProcess()

		t.Run(kv.k, func(subT *testing.T) {
			assert.Equal(subT, obj, Build(&r))
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

		r := NewReaderWithBuffers([]byte(objStr), &buffer, &charBuffer)
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
			v: JsonNumber{Value: []byte("222")},
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

	r := NewReaderWithBuffers([]byte(objStr), &buffer, &charBuffer)

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

		r := NewReaderWithBuffers([]byte(objStr), &buffer, &charBuffer)
		r.PreProcess()

		t.Run(fmt.Sprintf("json element with volume %d", s), func(subT *testing.T) {
			assert.Equal(subT, obj, BuildWithPartialDestruct(&r))
		})
	}
}

func BuildWithPartialDestruct(r *Reader) JsonElement {
	value := r.Any()
	switch value.Kind {
	case NumberValue:
		return JsonNumber{Value: value.Number.Value}
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
			return JsonNumber{Value: []byte(fmt.Sprintf("%d", rand.Int()%1000000))}
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
		return JsonNumber{Value: value.Number.Value}
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
