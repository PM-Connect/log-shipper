package nomad

import (
	"sync"

	nomadAPI "github.com/hashicorp/nomad/api"
)

type AllocationPool struct {
	sync.RWMutex
	Allocations []*nomadAPI.Allocation

	AllocationAdded chan *nomadAPI.Allocation
	AllocationRemoved chan *nomadAPI.Allocation
}

func NewAllocationPool() *AllocationPool {
	return &AllocationPool{
		AllocationAdded: make(chan *nomadAPI.Allocation),
		AllocationRemoved: make(chan *nomadAPI.Allocation),
	}
}

func (p *AllocationPool) Sync(allocations []*nomadAPI.Allocation) {
	for _, alloc := range allocations {
		if !p.Has(alloc) {
			p.AllocationAdded <- alloc
		}
	}

	p.RLock()
	for _, alloc := range p.Allocations {
		if !sliceContainsAllocation(alloc, allocations) {
			p.AllocationRemoved <- alloc
		}
	}
	p.RUnlock()

	p.Set(allocations)
}

func (p *AllocationPool) Set(allocations []*nomadAPI.Allocation) {
	p.Lock()
	p.Allocations = allocations
	p.Unlock()
}

func (p *AllocationPool) Has(allocation *nomadAPI.Allocation) bool {
	p.RLock()
	defer p.RUnlock()

	return sliceContainsAllocation(allocation, p.Allocations)
}

func sliceContainsAllocation(allocation *nomadAPI.Allocation, allocations []*nomadAPI.Allocation) bool {
	for _, i := range allocations {
		if i.ID == allocation.ID {
			return true
		}
	}

	return false
}