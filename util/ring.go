package util

import "sync/atomic"

// Ring rotates through items in a thread-safe way.
//
// It's intended for small fixed slices that are read concurrently.
type Ring[T any] struct {
	items []T
	idx   uint32
}

func NewRing[T any](items []T) *Ring[T] {
	// Copy to avoid unexpected mutation by callers.
	cp := append([]T(nil), items...)
	return &Ring[T]{items: cp}
}

func (r *Ring[T]) Next() (T, bool) {
	if r == nil {
		var zero T
		return zero, false
	}

	n := len(r.items)
	if n == 0 {
		var zero T
		return zero, false
	}

	for {
		old := atomic.LoadUint32(&r.idx)
		i := old % uint32(n)
		next := (old + 1) % uint32(n)
		if atomic.CompareAndSwapUint32(&r.idx, old, next) {
			return r.items[i], true
		}
	}
}

func (r *Ring[T]) Len() int {
	if r == nil {
		return 0
	}
	return len(r.items)
}
