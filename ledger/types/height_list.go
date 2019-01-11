package types

type HeightList struct {
	Heights []uint64
}

func (hl *HeightList) Append(height uint64) {
	hl.Heights = append(hl.Heights, height)
}
