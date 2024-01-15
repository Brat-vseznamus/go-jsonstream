package jreader

import (
	"bytes"
	"fmt"
)

const BufferConfigSeparator = "----"
const DataSeparator = "----"

const (
	ConstObject               byte = 'O'
	ConstObjectNeedPreprocess byte = 'U'
	ConstArray                byte = 'A'
	ConstNumber               byte = 'N'
	ConstString               byte = 'S'
)

func isBit(b byte) bool {
	return '0' == b || b == '1'
}

func Fuzz(data []byte) int {
	indexBuffer := bytes.Index(data, []byte(BufferConfigSeparator))
	correct := checkConfigDataCorrect(data, indexBuffer)
	if !correct {
		return -1
	}

	config := InitConfig(data)

	rest := data[indexBuffer+len(BufferConfigSeparator):]

	index := bytes.Index(rest, []byte(DataSeparator))
	if index == -1 {
		return -1
	}

	dataSequence := rest[:index]
	rawData := rest[index+len(DataSeparator):]

	for _, c := range dataSequence {
		switch c {
		case ConstObject, ConstObjectNeedPreprocess, ConstArray, ConstNumber, ConstString:
			continue
		default:
			return -1
		}
	}

	fmt.Printf("---- PREPROCESS CONFIG: %s ----\n", string(data[:indexBuffer]))
	fmt.Printf("---- EXPECTED SEQUENCE: %s ----\n", string(dataSequence))
	fmt.Printf("---- RAW DATA: %s ----\n", string(rawData))

	reader := NewReaderWithBuffers(rawData, config)
	seqIndex := 0

	return ParseJson(&reader, dataSequence, &seqIndex)
}

func checkConfigDataCorrect(data []byte, indexBuffer int) bool {
	if indexBuffer != 4 {
		return false
	}

	for i := 0; i < 4; i++ {
		if !isBit(data[i]) {
			return false
		}
	}
	return true
}

func InitConfig(data []byte) (config BufferConfig) {
	if data[0] == '1' {
		tmp := make([]JsonTreeStruct, 10)
		config.StructBuffer = &tmp
	}
	if data[1] == '1' {
		tmp := make([]byte, 10)
		config.CharsBuffer = &tmp
	}
	if data[2] == '1' {
		tmp := make([][]byte, 10)
		config.ComputedValuesBuffer.StringValues = &tmp
	}
	if data[3] == '1' {
		tmp := make([]NumberProps, 10)
		config.ComputedValuesBuffer.NumberValues = &tmp
	}
	return
}

func ParseJson(r *Reader, dataSequence []byte, seqIndex *int) (priority int) {
	if *seqIndex >= len(dataSequence) {
		return
	}
	seqElem := dataSequence[*seqIndex]
	switch seqElem {
	case ConstObject, ConstObjectNeedPreprocess:
		preProcessed := r.IsPreProcessed()
		if seqElem == ConstObjectNeedPreprocess && !preProcessed {
			r.PreProcess()
		}
		obj := r.Object()
		if r.Error() != nil {
			return
		}
		for obj.Next() {
			*seqIndex += 1
			result := ParseJson(r, dataSequence, seqIndex)
			if result == 0 {
				return
			}
		}
		if seqElem == ConstObjectNeedPreprocess && !preProcessed {
			r.SyncWithPreProcess()
		}
	case ConstArray:
		arr := r.Array()
		if r.Error() != nil {
			return
		}
		for arr.Next() {
			*seqIndex += 1
			result := ParseJson(r, dataSequence, seqIndex)
			if result == 0 {
				return
			}
		}
	case ConstString:
		r.String()
		if r.Error() != nil {
			return
		}
	case ConstNumber:
		r.NumberProps()
		if r.Error() != nil {
			return
		}
	}

	priority = 1
	return
}
