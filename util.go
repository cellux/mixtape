package main

import (
	"sync"
)

type Box[T any] struct {
	mu sync.Mutex
	v  T
}

func (box *Box[T]) Get() T {
	box.mu.Lock()
	defer box.mu.Unlock()
	return box.v
}

func (box *Box[T]) Set(v T) {
	box.mu.Lock()
	defer box.mu.Unlock()
	box.v = v
}
