package jreader

import "fmt"

// NewReader creates a Reader that consumes the specified JSON input data.
//
// This function returns the struct by value (Reader, not *Reader). This avoids the overhead of a
// heap allocation since, in typical usage, the Reader will not escape the scope in which it was
// declared and can remain on the stack.
func NewReader(data []byte) Reader {
	buffer := make([]JsonTreeStruct, 0)
	charBuffer := make([]byte, 0)
	return NewReaderWithBuffers(
		data,
		BufferConfig{
			StructBuffer:         &buffer,
			CharsBuffer:          &charBuffer,
			ComputedValuesBuffer: JsonComputedValues{},
		},
	)
}

func NewReaderWithBuffers(data []byte, bufferConfig BufferConfig) Reader {
	r := Reader{
		tr: newTokenReader(
			data,
			bufferConfig.StructBuffer,
			bufferConfig.CharsBuffer,
			bufferConfig.ComputedValuesBuffer,
		),
	}
	// temporary solution
	r.tr.options.readRawNumbers = true
	if bufferConfig.CharsBuffer == nil {
		r.err = fmt.Errorf("char buffer must be initilized")
	}
	return r
}

type BufferConfig struct {
	StructBuffer         *[]JsonTreeStruct
	CharsBuffer          *[]byte
	ComputedValuesBuffer JsonComputedValues
}
