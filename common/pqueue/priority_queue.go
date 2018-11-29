package pqueue

import (
	"container/heap"
	"fmt"
	"math/big"
)

//
// Element represents an element in the priority queue
//
type Element interface {
	Priority() *big.Int
	GetIndex() int
	SetIndex(index int)
}

//
// ElementList implements heap.Interface and holds Items.
//
type ElementList []Element

func (el ElementList) Len() int { return len(el) }

func (el ElementList) Less(i, j int) bool {
	return el[i].Priority().Cmp(el[j].Priority()) >= 0
}

func (el ElementList) Swap(i, j int) {
	el[i], el[j] = el[j], el[i]
	el[i].SetIndex(i)
	el[j].SetIndex(j)
}

func (el *ElementList) IsEmpty() bool {
	return (*el).Len() == 0
}

func (el *ElementList) Push(x interface{}) {
	n := len(*el)
	elem := x.(Element)
	elem.SetIndex(n)
	*el = append(*el, elem)
}

func (el *ElementList) Pop() interface{} {
	old := *el
	n := len(old)
	elem := old[n-1]
	elem.SetIndex(-1) // for safety
	*el = old[0 : n-1]
	return elem
}

func (el *ElementList) Peek() interface{} {
	return (*el)[0]
}

//
// PriorityQueue models a priority queue (max queue).
// The Pop() method returns the element with the MAX priority value
//
type PriorityQueue struct {
	elemList *ElementList
}

func CreatePriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{
		elemList: &ElementList{},
	}
	heap.Init(pq.elemList)
	return pq
}

func (pq *PriorityQueue) NumElements() int {
	numElems := pq.elemList.Len()
	return numElems
}

func (pq *PriorityQueue) ElementList() *ElementList {
	return pq.elemList
}

func (pq *PriorityQueue) IsEmpty() bool {
	return pq.elemList.Len() == 0
}

func (pq *PriorityQueue) Push(elem Element) {
	heap.Push(pq.elemList, elem)
}

func (pq *PriorityQueue) Pop() Element {
	elem := (heap.Pop(pq.elemList)).(Element)
	return elem
}

func (pq *PriorityQueue) Peek() Element {
	return pq.elemList.Peek().(Element)
}

func (pq *PriorityQueue) Remove(index int) error {
	numElems := pq.elemList.Len()
	if index >= numElems {
		return fmt.Errorf("index out of bound -- index: %v, number of elements: %v", index, numElems)
	}
	heap.Remove(pq.elemList, index)
	return nil
}
