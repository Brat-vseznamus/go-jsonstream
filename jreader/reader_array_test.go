package jreader

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddErrorStopsArrayParsing(t *testing.T) {
	r := NewReader([]byte("[1,2]"))
	arr := r.Array()
	require.True(t, arr.Next())
	require.Equal(t, int64(1), r.Int64())

	err := errors.New("sorry")
	r.AddError(err)
	require.Equal(t, err, r.Error())

	require.False(t, arr.Next())
	require.Equal(t, int64(0), r.Int64())
	require.Equal(t, err, r.Error())
}

func TestSyntaxErrorStopsArrayParsing(t *testing.T) {
	r := NewReader([]byte("[bad,1,2]"))
	arr := r.Array()
	require.False(t, arr.Next())
	require.Equal(t, int64(0), r.Int64())
	require.Error(t, r.Error())
}
