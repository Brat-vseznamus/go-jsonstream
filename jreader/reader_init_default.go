package jreader

// NewReader creates a Reader that consumes the specified JSON input data.
//
// This function returns the struct by value (Reader, not *Reader). This avoids the overhead of a
// heap allocation since, in typical usage, the Reader will not escape the scope in which it was
// declared and can remain on the stack.
func NewReader(data []byte) Reader {
	buffer := make([]JsonTreeStruct, 0)
	charBuffer := make([]byte, 0)
	computedValuesBuffer := JsonComputedValues{}
	return Reader{
		tr: newTokenReader(data, &buffer, &charBuffer, computedValuesBuffer),
	}
}

func NewReaderWithBuffers(data []byte, bufferConfig BufferConfig) Reader {
	return Reader{
		tr: newTokenReader(
			data,
			bufferConfig.StructBuffer,
			bufferConfig.CharsBuffer,
			bufferConfig.ComputedValuesBuffer,
		),
	}
}

type BufferConfig struct {
	StructBuffer         *[]JsonTreeStruct
	CharsBuffer          *[]byte
	ComputedValuesBuffer JsonComputedValues
}
