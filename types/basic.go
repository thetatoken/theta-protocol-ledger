package types

import (
	"fmt"
)

// Bytes represents bytes type.
type Bytes []byte

func (b Bytes) String() string {
	return fmt.Sprintf("%X", []byte(b))
}
