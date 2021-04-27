package core

import (
	"testing"

	"github.com/thetatoken/theta/common"
)

func TestEliteAggregateVote(t *testing.T) {
	vote := NewAggregatedEENVotes(common.HexToHash("deadbeef"))
	// if res := vote.Validate(); !res.IsError() || res.Message != "aggregated vote is empty" {
	// 	t.Fatal("Should fail empty vote:", res.Message)
	// }

	vote.Addresses = append(vote.Addresses, common.HexToAddress("a1"))
	// if res := vote.Validate(); !res.IsError() || res.Message != "aggregate vote lengths are inconsisent" {
	// 	t.Fatal("Should fail inconsistent vote:", res.Message)
	// }

	vote.Multiplies = append(vote.Multiplies, 0)
	// if res := vote.Validate(); !res.IsError() || res.Message != "aggregate vote lengths are inconsisent" {
	// 	t.Fatal("Should fail inconsistent vote:", res.Message)
	// }

	vote.Multiplies[0] = 1
	// if res := vote.Validate(); !res.IsOK() {
	// 	t.Fatal("Should pass:", res.Message)
	// }

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

func TestEliteAggregateVoteMerge(t *testing.T) {
	blockHash := common.HexToHash("deadbeef")

	// vote: [a1];
	vote := NewAggregatedEENVotes(blockHash)
	vote.Addresses = append(vote.Addresses, common.HexToAddress("a1"))
	vote.Multiplies = append(vote.Multiplies, 1)

	// vote: [a1, c3]
	vote.Addresses = append(vote.Addresses, common.HexToAddress("c3"))
	vote.Multiplies = append(vote.Multiplies, 5)

	// vote: [a1, c3]; vote2: [a1]
	vote2 := NewAggregatedEENVotes(blockHash)
	vote2.Addresses = append(vote2.Addresses, common.HexToAddress("a1"))
	vote2.Multiplies = append(vote2.Multiplies, 3)

	// vote: [a1, c3]; vote2: [a1, b2]
	vote2.Addresses = append(vote2.Addresses, common.HexToAddress("b2"))
	vote2.Multiplies = append(vote2.Multiplies, 2)

	// Merge vote and vote2
	aggv, err := vote.Merge(vote2)
	if aggv == nil {
		t.Fatalf("Should merge vote, err: %v, aggv: %v", err, aggv)
	}

	t.Logf("vote   : %v", vote)
	t.Logf("vote2  : %v", vote2)
	t.Logf("aggVote: %v", aggv)

	if aggv.Block != blockHash {
		t.Fatalf("The aggregated vote has an invalid block hash: %v", aggv)
	}

	if len(aggv.Multiplies) != 3 {
		t.Fatalf("The aggregated vote should have three multiplies: %v", aggv)
	}

	if aggv.Multiplies[0] != 4 || aggv.Multiplies[1] != 2 || aggv.Multiplies[2] != 5 {
		t.Fatalf("The aggregated vote has incorrect multiplies: %v", aggv)
	}

	if len(aggv.Addresses) != 3 {
		t.Fatalf("The aggregated vote should have three addresses: %v", aggv)
	}

	if aggv.Addresses[0] != vote.Addresses[0] || aggv.Addresses[0] != vote2.Addresses[0] || aggv.Addresses[1] != vote2.Addresses[1] || aggv.Addresses[2] != vote.Addresses[1] {
		t.Fatalf("Should merge vote and keep order: %v", aggv)
	}

}

func TestEliteAggregateVoteMergeSubset(t *testing.T) {
	blockHash := common.HexToHash("deadbeef")

	// vote: [a1];
	vote := NewAggregatedEENVotes(blockHash)
	vote.Addresses = append(vote.Addresses, common.HexToAddress("a1"))
	vote.Multiplies = append(vote.Multiplies, 1)

	// vote: [a1, c3]
	vote.Addresses = append(vote.Addresses, common.HexToAddress("c3"))
	vote.Multiplies = append(vote.Multiplies, 5)

	// vote: [a1, c3]; vote2: [a1]
	vote2 := NewAggregatedEENVotes(blockHash)
	vote2.Addresses = append(vote2.Addresses, common.HexToAddress("a1"))
	vote2.Multiplies = append(vote2.Multiplies, 3)

	// vote: [a1, c3]; vote2: [a1, b2]
	vote2.Addresses = append(vote2.Addresses, common.HexToAddress("b2"))
	vote2.Multiplies = append(vote2.Multiplies, 2)

	// vote: [a1, c3]; vote2: [a1, b2, c3]
	vote2.Addresses = append(vote2.Addresses, common.HexToAddress("c3"))
	vote2.Multiplies = append(vote2.Multiplies, 1)

	// vote: [a1, c3]; vote2: [a1, b2, c3, d4]
	vote2.Addresses = append(vote2.Addresses, common.HexToAddress("d4"))
	vote2.Multiplies = append(vote2.Multiplies, 5)

	// Merge vote and vote2
	aggv, err := vote.Merge(vote2)
	if aggv == nil {
		t.Fatalf("Should merge vote, err: %v, aggv: %v", err, aggv)
	}

	t.Logf("vote   : %v", vote)
	t.Logf("vote2  : %v", vote2)
	t.Logf("aggVote: %v", aggv)

	if aggv.Block != blockHash {
		t.Fatalf("The aggregated vote has an invalid block hash: %v", aggv)
	}

	if len(aggv.Multiplies) != 4 {
		t.Fatalf("The aggregated vote should have four multiplies: %v", aggv)
	}

	if aggv.Multiplies[0] != 4 || aggv.Multiplies[1] != 2 || aggv.Multiplies[2] != 6 || aggv.Multiplies[3] != 5 {
		t.Fatalf("The aggregated vote has incorrect multiplies: %v", aggv)
	}

	if len(aggv.Addresses) != 4 {
		t.Fatalf("The aggregated vote should have four addresses: %v", aggv)
	}

	if aggv.Addresses[0] != vote.Addresses[0] || aggv.Addresses[0] != vote2.Addresses[0] || aggv.Addresses[1] != vote2.Addresses[1] || aggv.Addresses[2] != vote.Addresses[1] || aggv.Addresses[3] != vote2.Addresses[3] {
		t.Fatalf("Should merge vote and keep order: %v", aggv)
	}

	aggv2, err := vote2.Merge(vote)
	if err != nil {
		t.Fatalf("Should merge, err: %v", err)
	}
	if aggv2 != nil {
		t.Fatalf("Should not create a new aggreated vote since vote is a subset of vote2, aggv: %v", aggv2)
	}
}
