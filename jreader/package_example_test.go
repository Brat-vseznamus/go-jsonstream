package jreader

import "fmt"

func Example() {
	r := NewReader([]byte(`"a \"good\" string"`))

	s := r.String()

	if err := r.Error(); err != nil {
		fmt.Println("error:", err.Error())
	} else {
		fmt.Println(string(s))
	}

	// Output: a \"good\" string
}

func ExampleWithEscapes() {
	charsBuffer := make([]byte, 10)
	stringsBuffer := make([][]byte, 10)

	r := NewReaderWithBuffers([]byte(`"a \"good\" string"`), BufferConfig{
		CharsBuffer: &charsBuffer,
		ComputedValuesBuffer: JsonComputedValues{
			StringValues: &stringsBuffer,
		},
	})

	s := r.String()

	if err := r.Error(); err != nil {
		fmt.Println("error:", err.Error())
	} else {
		fmt.Println(string(s))
	}

	// Output: a "good" string
}
