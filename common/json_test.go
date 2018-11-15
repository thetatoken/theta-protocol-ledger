package common

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Uint64Wrap struct {
	Age JSONUint64
}

func TestJSONUint64(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	var a JSONUint64 = 123
	s, err := json.Marshal(a)
	require.Nil(err)
	assert.Equal([]byte("\"123\""), s)

	var b JSONUint64
	(&b).UnmarshalText([]byte("123"))
	assert.Equal(JSONUint64(123), b)

	var c JSONUint64
	err = json.Unmarshal([]byte("123"), &c)
	assert.NotNil(err)

	json.Unmarshal([]byte("\"123\""), &c)
	assert.Equal(JSONUint64(123), c)

	e, err := json.Marshal(JSONUint64(math.MaxUint64))
	require.Nil(err)
	assert.Equal([]byte("\"18446744073709551615\""), e)

	var f JSONUint64
	err = json.Unmarshal([]byte("\"18446744073709551615\""), &f)
	fmt.Println(err)
	require.Nil(err)
	assert.Equal(JSONUint64(math.MaxUint64), f)

	x := Uint64Wrap{Age: 18446744073709551615}
	y, err := json.Marshal(x)
	require.Nil(err)
	var z Uint64Wrap
	err = json.Unmarshal(y, &z)
	require.Nil(err)

	assert.Equal(JSONUint64(math.MaxUint64), z.Age)
}

type BigWrap struct {
	Age *JSONBig
}

func TestJSONBig(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	var c JSONBig
	err := json.Unmarshal([]byte("123"), &c)
	assert.NotNil(err)
	err = json.Unmarshal([]byte("\"123\""), &c)
	assert.Nil(err)
	assert.Equal(int64(123), c.ToInt().Int64())

	num, ok := new(big.Int).SetString("1231231231231231231231231231321231231231231312313213123", 10)
	require.True(ok)
	x := BigWrap{Age: (*JSONBig)(num)}
	y, err := json.Marshal(x)
	require.Nil(err)
	var z BigWrap
	err = json.Unmarshal(y, &z)
	require.Nil(err)

	assert.Equal(0, num.Cmp((*big.Int)(z.Age)))
}
