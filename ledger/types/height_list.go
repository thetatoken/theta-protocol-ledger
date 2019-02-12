package types

import "fmt"

type HeightList struct {
	Heights []uint64
}

func (hl *HeightList) Append(height uint64) {
	hl.Heights = append(hl.Heights, height)
}

func (hl *HeightList) Contains(height uint64) bool {
	for _, h := range hl.Heights {
		if h == height {
			return true
		}
	}
	return false
}

func (hl *HeightList) String() string {
	return fmt.Sprintf("{HeightList: %v}", hl.Heights)
}
