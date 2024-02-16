package jreader

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"strings"
	"testing"
	"time"
)

const SIZE = 1024 * 1024
const N = 200

var Index = 0

func bruteForceFind(arr []byte, sep byte) int {
	for i, b := range arr {
		//Index += 1
		//bytes.IndexAny()
		if sep == b {
			return i
		}
	}
	return -1
}

func TestFindByte(t *testing.T) {
	for i := 1; i <= SIZE; i <<= 1 {
		avg1, avg2 := 0.0, 0.0
		for j := 0; j < N; j++ {
			s := make([]byte, i)
			for c := 0; c < i; c++ {
				s[c] = byte(rand.Int()%20) + 'A'
			}
			s[rand.Int()%i] = '*'

			s1 := time.Now()
			i1 := bruteForceFind(s, '*')
			d1 := time.Since(s1)
			avg1 += float64(d1.Nanoseconds()) / float64(N)

			s2 := time.Now()
			i2 := bytes.IndexByte(s, '*')
			d2 := time.Since(s2)
			avg2 += float64(d2.Nanoseconds()) / float64(N)

			assert.Equal(t, i1, i2)
		}

		savg1 := fmt.Sprintf("%f", avg1)
		savg1 = savg1[0:strings.Index(savg1, ".")] + "," + savg1[strings.Index(savg1, ".")+1:]

		savg2 := fmt.Sprintf("%f", avg2)
		savg2 = savg2[0:strings.Index(savg2, ".")] + "," + savg2[strings.Index(savg2, ".")+1:]

		fmt.Printf("%d; %s; %s\n", i, savg1, savg2)
		//fmt.Printf("AVG Time on size %d: %f, %f nanosec\n", i, avg1, avg2)
	}
}

func TestCrashes(t *testing.T) {
	Fuzz([]byte("0000----AU----[{}]"))
}

func TestAddErrorStopsObjectParsing(t *testing.T) {
	r := NewReader([]byte(`{"a":1, "b":2}`))
	obj := r.Object()
	require.True(t, obj.Next())
	require.Equal(t, "a", string(obj.Name()))
	require.Equal(t, 1, r.Int64())

	err := errors.New("sorry")
	r.AddError(err)
	require.Equal(t, err, r.Error())

	require.False(t, obj.Next())
	require.Equal(t, err, r.Error())
}

func TestSyntaxErrorStopsObjectParsing(t *testing.T) {
	r := NewReader([]byte(`{"a":1, x: 2, "c":3}`))
	obj := r.Object()
	require.True(t, obj.Next())
	require.Equal(t, "a", string(obj.Name()))
	require.Equal(t, 1, r.Int64())

	require.False(t, obj.Next())
	require.Equal(t, 0, r.Int64())

	require.Error(t, r.Error())

	require.False(t, obj.Next())
}
