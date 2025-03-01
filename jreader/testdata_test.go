package jreader

import (
	"github.com/Brat-vseznamus/go-jsonstream/v3/internal/commontest"
)

// ExampleStruct is defined in another package, so we need to wrap it in our own type to define methods on it.
type ExampleStructWrapper commontest.ExampleStruct

func (s *ExampleStructWrapper) ReadFromJSONReader(r *Reader) {
	for obj := r.Object(); obj.Next(); {
		switch string(obj.Name()) {
		case commontest.ExampleStructStringFieldName:
			s.StringField = string(r.String())
		case commontest.ExampleStructIntFieldName:
			s.IntField = r.Int64()
		case commontest.ExampleStructOptBoolAsInterfaceFieldName:
			b, nonNull := r.BoolOrNull()
			if nonNull {
				s.OptBoolAsInterfaceField = b
			} else {
				s.OptBoolAsInterfaceField = nil
			}
		}
	}
}
