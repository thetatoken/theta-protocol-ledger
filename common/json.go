package common

import (
	"errors"
	"math/big"
	"strconv"

	"github.com/thetatoken/theta/common/hexutil"
)

type JSONBig big.Int

// MarshalText implements encoding.TextMarshaler
func (b JSONBig) MarshalText() ([]byte, error) {
	return []byte((*big.Int)(&b).String()), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (b *JSONBig) UnmarshalJSON(input []byte) error {
	if !hexutil.IsString(input) {
		return errors.New("Big number must be formatted as string")
	}
	return b.UnmarshalText(input[1 : len(input)-1])
}

// UnmarshalText implements encoding.TextUnmarshaler
func (b *JSONBig) UnmarshalText(input []byte) error {
	_, ok := (*big.Int)(b).SetString(string(input), 10)
	if !ok {
		return errors.New("Failed to parse big.Int")
	}
	return nil
}

// ToInt converts b to a big.Int.
func (b *JSONBig) ToInt() *big.Int {
	return (*big.Int)(b)
}

type JSONUint64 uint64

// MarshalText implements encoding.TextMarshaler.
func (b JSONUint64) MarshalText() ([]byte, error) {
	buf := strconv.AppendUint([]byte{}, uint64(b), 10)
	return buf, nil
}

// UnmarshalText implements encoding.TextUnmarshaler
func (b *JSONUint64) UnmarshalText(raw []byte) error {
	res, err := strconv.ParseUint(string(raw), 10, 64)
	if err != nil {
		return err
	}
	*b = JSONUint64(res)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (b *JSONUint64) UnmarshalJSON(input []byte) error {
	if !hexutil.IsString(input) {
		return errors.New("Uint64 must be formatted as string")
	}
	return b.UnmarshalText(input[1 : len(input)-1])
}
