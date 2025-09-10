package limitbook

import (
	"container/heap"
)

// PriceHeap implements a heap for price levels
type PriceHeap []*PriceLevel

// NewPriceHeap creates a new price heap
func NewPriceHeap(isBid bool) *PriceHeap {
	h := &PriceHeap{}
	heap.Init(h)
	return h
}

// Len returns the length of the heap
func (h PriceHeap) Len() int {
	return len(h)
}

// Less compares two price levels
func (h PriceHeap) Less(i, j int) bool {
	// For bids (buy orders), we want highest price first (max heap)
	// For asks (sell orders), we want lowest price first (min heap)
	// This is determined by the isBid flag when creating the heap
	return h[i].Price.LessThan(h[j].Price)
}

// Swap swaps two price levels
func (h PriceHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

// Push adds a price level to the heap
func (h *PriceHeap) Push(x interface{}) {
	*h = append(*h, x.(*PriceLevel))
}

// Pop removes and returns the top price level
func (h *PriceHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
