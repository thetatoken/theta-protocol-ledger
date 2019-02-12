package types

import "strconv"

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

func (hl *HeightList) JsonString() string {
	str := "["
	for i, height := range hl.Heights {
		str += strconv.FormatUint(height, 10)
		if i < len(hl.Heights)-1 {
			str += ","
		}
	}
	str += "]"
	return str
}
