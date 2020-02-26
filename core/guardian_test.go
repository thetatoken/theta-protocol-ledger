package core

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto/bls"
	"github.com/thetatoken/theta/rlp"

	"github.com/thetatoken/theta/crypto"
)

func createTestGuardianPool(size int) (*GuardianCandidatePool, map[common.Address]*bls.SecretKey) {
	pool := NewGuardianCandidatePool()
	sks := make(map[common.Address]*bls.SecretKey)
	for i := 0; i < size; i++ {
		_, pub, _ := crypto.GenerateKeyPair()
		blsKey, _ := bls.RandKey()
		g := &Guardian{
			StakeHolder: &StakeHolder{
				Holder: pub.Address(),
				Stakes: []*Stake{&Stake{
					Source:       pub.Address(),
					Amount:       MinGuardianStakeDeposit,
					Withdrawn:    false,
					ReturnHeight: 99999999999,
				}},
			},
			Pubkey: blsKey.PublicKey(),
		}
		pool.Add(g)
		sks[g.Holder] = blsKey
	}
	return pool, sks
}

func isSorted(pl *GuardianCandidatePool) bool {
	g := pl.SortedGuardians[0]
	for i := 1; i < pl.Len(); i++ {
		if bytes.Compare(g.Holder.Bytes(), pl.SortedGuardians[i].Holder.Bytes()) >= 0 {
			return false
		}
	}
	return true
}

func TestGuardianPool(t *testing.T) {
	require := require.New(t)

	pool, _ := createTestGuardianPool(10)

	// Should be sorted.
	if !isSorted(pool) {
		t.Fatal("Guardian pool is not sorted")
	}

	// Should not add duplicate.
	newGuardian := &Guardian{
		StakeHolder: &StakeHolder{
			Holder: pool.SortedGuardians[3].Holder,
		},
	}
	if pool.Add(newGuardian) {
		t.Fatal("Should not add duplicate guardian")
	}

	// Should add new guardian.
	_, pub, _ := crypto.GenerateKeyPair()
	blsKey, _ := bls.RandKey()
	g := &Guardian{
		StakeHolder: &StakeHolder{
			Holder: pub.Address(),
			Stakes: []*Stake{&Stake{
				Source:       pub.Address(),
				Amount:       MinGuardianStakeDeposit,
				Withdrawn:    false,
				ReturnHeight: 99999999999,
			}},
		},
		Pubkey: blsKey.PublicKey(),
	}
	if !pool.Add(g) || pool.Len() != 11 {
		t.Fatal("Should add new guardian")
	}
	if !isSorted(pool) {
		t.Fatal("Should be sorted after add")
	}

	// Should remove guardian.
	toRemove := pool.SortedGuardians[5].Holder
	toRemoveBlsPub := pool.SortedGuardians[5].Pubkey
	if !pool.Remove(toRemove) || pool.Len() != 10 {
		t.Fatal("Should remove guardian")
	}
	if !isSorted(pool) {
		t.Fatal("Should be sorted after remove")
	}

	// Should return false when removing non-existent guardian.
	if pool.Remove(toRemove) || pool.Len() != 10 {
		t.Fatal("Should not remove non-existent guardian")
	}

	// Should return -1 for removed guardian.
	require.Equal(-1, pool.Index(toRemoveBlsPub), "Should return -1 for removed guardian")

	toWithdrawnPub := pool.SortedGuardians[3].Pubkey
	nextPub := pool.SortedGuardians[4].Pubkey
	require.Equal(3, pool.WithStake().Index(toWithdrawnPub))
	require.Equal(4, pool.WithStake().Index(nextPub))
	pool.SortedGuardians[3].Stakes[0].Withdrawn = true
	// Should return -1 for withdrawn guardian.
	require.Equal(-1, pool.WithStake().Index(toWithdrawnPub))
	// Should skip withdrawn guardian.
	require.Equal(3, pool.WithStake().Index(nextPub))
}

func TestAggregateVote(t *testing.T) {
	pool, sks := createTestGuardianPool(10)

	bh := common.BytesToHash([]byte{12})
	vote1 := NewAggregateVotes(bh, pool)

	g1 := pool.SortedGuardians[0].Holder

	// Guardian 1 signs a vote.
	success := vote1.Sign(sks[g1], 0)
	if !success {
		t.Fatal("Should sign")
	}
	if res := vote1.Validate(pool); res.IsError() {
		t.Fatal("Should validate", res.Message)
	}

	// Guardian 2 signs a vote.
	vote2 := NewAggregateVotes(bh, pool)
	g2 := pool.SortedGuardians[1].Holder
	success = vote2.Sign(sks[g2], 1)
	if !success {
		t.Fatal("Should sign")
	}
	if res := vote2.Validate(pool); res.IsError() {
		t.Fatal("Should validate", res.Message)
	}

	// Should merge two votes.
	vote12, err := vote1.Merge(vote2)
	if err != nil {
		t.Fatalf("Failed to merge votes: %s", err.Error())
	}
	if res := vote12.Validate(pool); res.IsError() {
		t.Fatal("Should validate", res.Message)
	}

	// Should not merge votes that is a subset of current vote.
	res, err := vote12.Merge(vote2)
	if err != nil || res != nil {
		t.Fatalf("Should not merge votes that is subset")
	}
	res, err = vote12.Merge(NewAggregateVotes(bh, pool))
	if err != nil || res != nil {
		t.Fatalf("Should not merge votes that is subset")
	}
	res, err = vote12.Merge(vote12)
	if err != nil || res != nil {
		t.Fatalf("Should not merge votes that is subset")
	}
}

func TestAggregateVoteEncoding(t *testing.T) {
	require := require.New(t)

	pool, sks := createTestGuardianPool(10)

	bh := common.BytesToHash([]byte{12})
	vote1 := NewAggregateVotes(bh, pool)

	g1 := pool.SortedGuardians[0].Holder

	// Guardian 1 signs a vote.
	success := vote1.Sign(sks[g1], 0)
	require.True(success, "Should sign")

	raw, err := rlp.EncodeToBytes(vote1)
	require.Nil(err)

	vote2 := &AggregatedVotes{}
	err = rlp.DecodeBytes(raw, vote2)
	require.Nil(err)
}
