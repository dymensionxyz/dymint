package queue

import (
	"fmt"
	"strings"
)

type Queue[T any] struct {
	elements []T
}

func FromSlice[T any](s []T) *Queue[T] {
	return &Queue[T]{elements: s}
}

func New[T any]() *Queue[T] {
	return &Queue[T]{elements: make([]T, 0)}
}

func (q *Queue[T]) Enqueue(values ...T) {
	q.elements = append(q.elements, values...)
}

func (q *Queue[T]) DequeueAll() []T {
	values := q.elements
	q.elements = make([]T, 0)
	return values
}

func (q *Queue[T]) Size() int {
	return len(q.elements)
}

func (q *Queue[T]) String() string {
	str := "Queue["
	values := []string{}
	for _, value := range q.elements {
		values = append(values, fmt.Sprintf("%v", value))
	}
	str += strings.Join(values, ", ") + "]"
	return str
}
