package itertools

import "sync"

type queue[T any] struct {
	s  []T
	mu sync.RWMutex
}

func (q *queue[T]) Append(v T) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.s = append(q.s, v)
}

func (q *queue[T]) PopLeft() T {
	q.mu.Lock()
	defer q.mu.Unlock()

	v := q.s[0]
	q.s = q.s[1:]
	return v
}

func (q *queue[T]) IsEmpty() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.s) == 0
}
