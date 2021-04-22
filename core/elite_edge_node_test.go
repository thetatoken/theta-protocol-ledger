package core

import (
	"testing"

	"github.com/thetatoken/theta/common"
)

func TestEliteAggregateVote(t *testing.T) {
	vote := NewAggregatedEENVotes(common.HexToHash("deadbeef"))
	if res := vote.Validate(); !res.IsError() || res.Message != "aggregated vote is empty" {
		t.Fatal("Should fail empty vote:", res.Message)
	}

	vote.Addresses = append(vote.Addresses, common.HexToAddress("a1"))
	if res := vote.Validate(); !res.IsError() || res.Message != "aggregate vote lengths are inconsisent" {
		t.Fatal("Should fail inconsistent vote:", res.Message)
	}

	vote.Multiplies = append(vote.Multiplies, 0)
	if res := vote.Validate(); !res.IsError() || res.Message != "aggregate vote lengths are inconsisent" {
		t.Fatal("Should fail inconsistent vote:", res.Message)
	}

	vote.Multiplies[0] = 1
	if res := vote.Validate(); !res.IsOK() {
		t.Fatal("Should pass:", res.Message)
	}

	vote2 := NewAggregatedEENVotes(common.HexToHash("ff"))
	if _, err := vote.Merge(vote2); err == nil || err.Error() != "Cannot merge incompatible votes" {
		t.Fatal("Should not merge incompatible vote:", err)
	}

	vote2 = NewAggregatedEENVotes(common.HexToHash("deadbeef"))
	if aggregatedVote, err := vote.Merge(vote2); aggregatedVote != nil {
		t.Fatal("Should not empty vote:", err)
	}

	// vote: [a1]; vote2: [a1]
	vote2 = NewAggregatedEENVotes(common.HexToHash("deadbeef"))
	vote2.Addresses = append(vote2.Addresses, common.HexToAddress("a1"))
	vote2.Multiplies = append(vote2.Multiplies, 3)
	if aggregatedVote, err := vote.Merge(vote2); aggregatedVote != nil {
		t.Fatal("Should not merge subset vote:", err, aggregatedVote)
	}

	// vote: [a1, b2]; vote2: [a1]
	vote.Addresses = append(vote.Addresses, common.HexToAddress("b2"))
	vote.Multiplies = append(vote.Multiplies, 3)
	if aggregatedVote, err := vote.Merge(vote2); aggregatedVote != nil {
		t.Fatal("Should not merge subset vote:", err, aggregatedVote)
	}

	// vote: [a1, b2]; vote2: [a1, c3]
	vote2.Addresses = append(vote.Addresses, common.HexToAddress("c3"))
	vote2.Multiplies = append(vote.Multiplies, 3)
	if aggregatedVote, err := vote.Merge(vote2); aggregatedVote == nil {
		t.Fatal("Should merge vote", err, aggregatedVote)
	}

	// vote: [a1, b2]; vote2: [a1, b2, c3]
	vote2.Addresses = []common.Address{common.HexToAddress("a1"), common.HexToAddress("b2"), common.HexToAddress("c3")}
	vote2.Multiplies = []uint32{1, 2, 3}
	if aggregatedVote, err := vote.Merge(vote2); aggregatedVote == nil {
		t.Fatal("Should merge vote", err, aggregatedVote)
	}

	// should keep order
	v1 := NewAggregatedEENVotes(common.HexToHash("deadbeef"))
	v2 := NewAggregatedEENVotes(common.HexToHash("deadbeef"))
	v3 := NewAggregatedEENVotes(common.HexToHash("deadbeef"))
	v1.Addresses = []common.Address{common.HexToAddress("a1")}
	v1.Multiplies = []uint32{1}
	v2.Addresses = []common.Address{common.HexToAddress("b2")}
	v2.Multiplies = []uint32{2}
	v3.Addresses = []common.Address{common.HexToAddress("c3")}
	v3.Multiplies = []uint32{3}

	v, _ := v3.Merge(v1)
	v, _ = v.Merge(v2)
	if v == nil || len(v.Addresses) != 3 || v.Addresses[0] != v1.Addresses[0] || v.Addresses[1] != v2.Addresses[0] || v.Addresses[2] != v3.Addresses[0] {
		t.Fatal("Should merge vote and keep order", v)
	}
}
