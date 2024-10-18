package queue

import (
	"fmt"
	"strings"
)

// Queue holds elements in an array-list.
// This implementation is NOT thread-safe!
type Queue[T any] struct {
	elements []T
}

// FromSlice instantiates a new queue from the given slice.
func FromSlice[T any](s []T) *Queue[T] {
	return &Queue[T]{elements: s}
}

// New instantiates a new empty queue
func New[T any]() *Queue[T] {
	return &Queue[T]{elements: make([]T, 0)}
}

// Enqueue adds a value to the end of the queue
func (q *Queue[T]) Enqueue(values ...T) {
	q.elements = append(q.elements, values...)
}

// DequeueAll returns all queued elements (FIFO order) and cleans the entire queue.
func (q *Queue[T]) DequeueAll() []T {
	values := q.elements
	q.elements = make([]T, 0)
	return values
}

// Size returns number of elements within the queue.
func (q *Queue[T]) Size() int {
	return len(q.elements)
}

// String returns a string representation.
func (q *Queue[T]) String() string {
	str := "Queue["
	values := []string{}
	for _, value := range q.elements {
		values = append(values, fmt.Sprintf("%v", value))
	}
	str += strings.Join(values, ", ") + "]"
	return str
}
