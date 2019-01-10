package types

type HeightList struct {
	heights []uint64
}

func (hl *HeightList) Append(height uint64) {
	hl.heights = append(hl.heights, height)
}

func (hl *HeightList) Heights() []uint64 {
	return hl.heights
}
