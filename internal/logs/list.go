package logs

import (
	"container/list"
	"sync"
)

type EvictingList[T any] struct {
	data     *list.List
	capacity int
	mu       sync.Mutex
}

func NewEvictingList[T any](capacity int) *EvictingList[T] {
	return &EvictingList[T]{
		capacity: capacity,
		data:     list.New(),
	}
}

func (l *EvictingList[T]) Add(value T) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.capacity > 0 && l.data.Len() == l.capacity {
		l.data.Remove(l.data.Front())
	}
	l.data.PushBack(value)
}

func (l *EvictingList[T]) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.data.Len()
}

func (l *EvictingList[T]) Capacity() int {
	return l.capacity
}

func (l *EvictingList[T]) PopFront() (T, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.data.Len() == 0 {
		var zero T
		return zero, false
	}
	front := l.data.Front()
	val := front.Value.(T)
	l.data.Remove(front)
	return val, true
}

func (l *EvictingList[T]) Values() []T {
	l.mu.Lock()
	defer l.mu.Unlock()

	values := make([]T, 0, l.data.Len())
	for elem := l.data.Front(); elem != nil; elem = elem.Next() {
		values = append(values, elem.Value.(T))
	}
	return values
}
