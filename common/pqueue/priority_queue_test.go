package pqueue

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPriorityQueue(t *testing.T) {
	assert := assert.New(t)
	pq := initializePriorityQueue()
	checkElementOrder(pq, t, assert)
}

func TestPriorityRemoveElements(t *testing.T) {
	assert := assert.New(t)
	pq := initializePriorityQueue()
	numElems := pq.NumElements()

	index := numElems + 3
	err := pq.Remove(index)
	assert.NotNil(err)

	index = numElems
	err = pq.Remove(index)
	assert.NotNil(err)

	index = numElems - 1
	err = pq.Remove(index)
	assert.Nil(err)

	err = pq.Remove(2)
	assert.Nil(err)

	err = pq.Remove(4)
	assert.Nil(err)

	err = pq.Remove(5)
	assert.Nil(err)

	checkElementOrder(pq, t, assert)
}

func TestPriorityRemoveElements2(t *testing.T) {
	assert := assert.New(t)
	pq := initializePriorityQueue()

	prioritiesToRemove := []*big.Int{
		new(big.Int).SetUint64(19),
		new(big.Int).SetUint64(45),
		new(big.Int).SetUint64(7985),
	}
	t.Logf("----------------------------------")
	for _, priority := range prioritiesToRemove {
		t.Logf("Will remove: %v", priority)
	}
	t.Logf("----------------------------------")

	for _, priority := range prioritiesToRemove {
		elemList := *pq.ElementList()
		for idx, elem := range elemList {
			if elem.Priority().Cmp(priority) == 0 {
				pq.Remove(idx)
			}
		}
	}

	prevElemPriority := new(big.Int).SetUint64(99999999999)
	for {
		if !pq.IsEmpty() {
			elem := (pq.Pop()).(Element)
			currElemPriority := elem.Priority()
			t.Logf("elem priority: %v", currElemPriority)
			assert.True(prevElemPriority.Cmp(currElemPriority) >= 0)
			prevElemPriority = currElemPriority

			for _, priority := range prioritiesToRemove {
				assert.True(elem.Priority().Cmp(priority) != 0) // Should have been removed
			}
		} else {
			break
		}
	}
	t.Logf("----------------------------------")
}

// ----------- Test Utils ----------- //

func initializePriorityQueue() *PriorityQueue {
	pq := CreatePriorityQueue()

	pq.Push(createPQElem(5))
	pq.Push(createPQElem(8))
	pq.Push(createPQElem(19))
	pq.Push(createPQElem(3))
	pq.Push(createPQElem(1))
	pq.Push(createPQElem(2))
	pq.Push(createPQElem(45))
	pq.Push(createPQElem(45))
	pq.Push(createPQElem(145))
	pq.Push(createPQElem(425))
	pq.Push(createPQElem(33))
	pq.Push(createPQElem(31))
	pq.Push(createPQElem(18))
	pq.Push(createPQElem(7985))

	return pq
}

func checkElementOrder(pq *PriorityQueue, t *testing.T, assert *assert.Assertions) {
	prevElemPriority := new(big.Int).SetUint64(99999999999)
	for {
		if !pq.IsEmpty() {
			elem := (pq.Pop()).(Element)
			currElemPriority := elem.Priority()
			t.Logf("elem priority: %v", currElemPriority)
			assert.True(prevElemPriority.Cmp(currElemPriority) >= 0)
			prevElemPriority = currElemPriority
		} else {
			break
		}
	}
}

// pqelem implements the Element interface
type pqelem struct {
	priority *big.Int
	index    int
}

func createPQElem(priority uint64) *pqelem {
	return &pqelem{
		priority: new(big.Int).SetUint64(priority),
	}
}

func (pqe *pqelem) Priority() *big.Int {
	return pqe.priority
}

func (pqe *pqelem) SetIndex(index int) {
	pqe.index = index
}

func (pqe *pqelem) GetIndex() int {
	return pqe.index
}
